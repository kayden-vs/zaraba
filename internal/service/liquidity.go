package service

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/kayden-vs/zaraba/internal/engine"
	"github.com/kayden-vs/zaraba/pb"
)

// DemoLiquidityManager keeps the orderbook/trades populated on demand for demo usage.
// It only tops up when handlers call EnsureDemoLiquidity, so there is no always-on loop.
type DemoLiquidityManager struct {
	mu            sync.Mutex
	rng           *rand.Rand
	lastTopUp     time.Time
	lastSideBuy   bool
	minLevels     int
	spreadBps     float64
	baseSize      float64
	basePriceUSD  float64
	jitterPct     float64
	cooldown      time.Duration
	recenterBps   float64
	priceDriftBps float64
	priceTTL      time.Duration
	apiKey        string
	cachedPrice   int64
	cachedAt      time.Time
	enabled       bool
}

var demoLiquidity = newDemoLiquidityManager()

func newDemoLiquidityManager() *DemoLiquidityManager {
	return &DemoLiquidityManager{
		rng:           rand.New(rand.NewSource(time.Now().UnixNano())),
		minLevels:     envInt("DEMO_LIQ_MIN_LEVELS", 5),
		spreadBps:     envFloat("DEMO_LIQ_SPREAD_BPS", 12),
		baseSize:      envFloat("DEMO_LIQ_BASE_SIZE", 0.003),
		basePriceUSD:  envFloat("DEMO_LIQ_BASE_PRICE_USD", 74000),
		jitterPct:     envFloat("DEMO_LIQ_JITTER_PCT", 0.25),
		cooldown:      time.Duration(envInt("DEMO_LIQ_COOLDOWN_MS", 1500)) * time.Millisecond,
		recenterBps:   envFloat("DEMO_LIQ_RECENTER_BPS", 300),
		priceDriftBps: envFloat("DEMO_LIQ_PRICE_DRIFT_BPS", 8),
		priceTTL:      time.Duration(envInt("DEMO_LIQ_PRICE_TTL_MS", 15000)) * time.Millisecond,
		apiKey:        os.Getenv("API_KEY"),
		enabled:       envBool("DEMO_LIQ_ENABLED", true),
	}
}

func EnsureDemoLiquidity(server *ExchangeServer) {
	demoLiquidity.Ensure(server)
}

func EnsureDemoLiquidityForMarketOrder(server *ExchangeServer, bid bool, size int64) {
	demoLiquidity.EnsureForMarketOrder(server, bid, size)
}

func (m *DemoLiquidityManager) Ensure(server *ExchangeServer) {
	if !m.enabled || server == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if time.Since(m.lastTopUp) < m.cooldown {
		return
	}

	snapshot, err := server.StreamOrderBook(context.Background(), &pb.OrderBookRequest{Market: "demo"})
	if err != nil {
		log.Printf("demo liquidity: snapshot failed: %v", err)
		return
	}

	targetMid := m.targetMidPrice(snapshot)
	if targetMid <= 0 {
		targetMid = engine.PriceToInt(m.basePriceUSD)
	}

	bookMid := m.pickMidPrice(snapshot)
	if m.shouldRecenter(bookMid, targetMid) {
		server.ResetOrderbook()
		snapshot = &pb.OrderbookSnapshot{}
	}

	asks := len(snapshot.Asks)
	bids := len(snapshot.Bids)
	needAsks := m.minLevels - asks
	needBids := m.minLevels - bids
	midPrice := targetMid
	if bookMid > 0 {
		midPrice = bookMid
	}

	if needAsks <= 0 && needBids <= 0 {
		m.pulseFlow(server)
		if RecentTradesCount() == 0 {
			m.seedOneTradeTick(snapshot)
		}
		m.lastTopUp = time.Now()
		return
	}

	maxNeed := needAsks
	if needBids > maxNeed {
		maxNeed = needBids
	}

	for i := 0; i < maxNeed; i++ {
		offsetBps := m.spreadBps * float64(i+1)
		priceDelta := int64(math.Round(float64(midPrice) * (offsetBps / 10_000.0)))
		if priceDelta <= 0 {
			priceDelta = 1
		}

		size := m.nextSize()
		orderIDBase := time.Now().UnixNano() + int64(i*10)

		if i < needBids {
			price := midPrice - priceDelta
			if price > 0 {
				_, _ = server.PlaceLimitOrder(context.Background(), &pb.PlaceLimitOrderRequest{
					Price: price,
					Order: &pb.Order{
						Id:        orderIDBase + 1,
						Price:     price,
						Bid:       true,
						Size:      size,
						Timestamp: time.Now().UnixNano(),
					},
				})
			}
		}

		if i < needAsks {
			price := midPrice + priceDelta
			_, _ = server.PlaceLimitOrder(context.Background(), &pb.PlaceLimitOrderRequest{
				Price: price,
				Order: &pb.Order{
					Id:        orderIDBase + 2,
					Price:     price,
					Bid:       false,
					Size:      size,
					Timestamp: time.Now().UnixNano(),
				},
			})
		}
	}

	m.pulseFlow(server)

	if RecentTradesCount() == 0 {
		m.seedOneTradeTick(nil)
	}

	m.lastTopUp = time.Now()
}

func (m *DemoLiquidityManager) EnsureForMarketOrder(server *ExchangeServer, bid bool, size int64) {
	if !m.enabled || server == nil || size <= 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	snapshot, err := server.StreamOrderBook(context.Background(), &pb.OrderBookRequest{Market: "demo"})
	if err != nil {
		return
	}

	targetMid := m.targetMidPrice(snapshot)
	if targetMid <= 0 {
		targetMid = engine.PriceToInt(m.basePriceUSD)
	}

	bookMid := m.pickMidPrice(snapshot)
	if m.shouldRecenter(bookMid, targetMid) {
		server.ResetOrderbook()
		snapshot = &pb.OrderbookSnapshot{}
		bookMid = 0
	}

	midPrice := targetMid
	if bookMid > 0 {
		midPrice = bookMid
	}

	total := oppositeDepthVolume(snapshot, bid)
	if total >= size {
		return
	}

	missing := size - total
	m.injectDepthForSide(server, bid, missing, midPrice)
	m.lastTopUp = time.Now()
}

func oppositeDepthVolume(snapshot *pb.OrderbookSnapshot, bid bool) int64 {
	if snapshot == nil {
		return 0
	}

	var total int64
	if bid {
		for _, limit := range snapshot.Asks {
			total += limit.TotalVolume
		}
		return total
	}

	for _, limit := range snapshot.Bids {
		total += limit.TotalVolume
	}

	return total
}

func (m *DemoLiquidityManager) injectDepthForSide(server *ExchangeServer, bid bool, missing, midPrice int64) {
	if missing <= 0 || midPrice <= 0 {
		return
	}

	levels := maxInt(3, m.minLevels/2)
	remaining := missing

	for i := 0; i < levels && remaining > 0; i++ {
		offsetBps := m.spreadBps * float64(i+1)
		priceDelta := int64(math.Round(float64(midPrice) * (offsetBps / 10_000.0)))
		if priceDelta <= 0 {
			priceDelta = 1
		}

		chunk := remaining / int64(levels-i)
		if chunk <= 0 {
			chunk = 1
		}

		// Add mild variability so depth does not look perfectly synthetic.
		size := chunk + m.nextSize()/2
		if size <= 0 {
			size = chunk
		}

		orderBid := !bid
		price := midPrice - priceDelta
		if !orderBid {
			price = midPrice + priceDelta
		}
		if price <= 0 {
			continue
		}

		orderID := time.Now().UnixNano() + int64(i)
		_, _ = server.PlaceLimitOrder(context.Background(), &pb.PlaceLimitOrderRequest{
			Price: price,
			Order: &pb.Order{
				Id:        orderID,
				Price:     price,
				Bid:       orderBid,
				Size:      size,
				Timestamp: time.Now().UnixNano(),
			},
		})

		remaining -= chunk
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m *DemoLiquidityManager) shouldRecenter(bookMid, targetMid int64) bool {
	if bookMid <= 0 || targetMid <= 0 {
		return false
	}
	delta := math.Abs(float64(bookMid - targetMid))
	if delta <= 0 {
		return false
	}
	bps := delta / float64(targetMid) * 10_000
	return bps >= m.recenterBps
}

func (m *DemoLiquidityManager) targetMidPrice(snapshot *pb.OrderbookSnapshot) int64 {
	price := m.fetchReferencePrice()
	if price <= 0 {
		price = m.pickMidPrice(snapshot)
	}
	if price <= 0 {
		price = engine.PriceToInt(m.basePriceUSD)
	}

	if m.priceDriftBps > 0 {
		drift := (m.rng.Float64()*2 - 1) * m.priceDriftBps
		delta := int64(math.Round(float64(price) * (drift / 10_000.0)))
		price += delta
	}

	if price <= 0 {
		return engine.PriceToInt(m.basePriceUSD)
	}

	return price
}

func (m *DemoLiquidityManager) fetchReferencePrice() int64 {
	if m.cachedPrice > 0 && time.Since(m.cachedAt) < m.priceTTL {
		return m.cachedPrice
	}

	if price := m.fetchCoinGeckoSimple(); price > 0 {
		m.cachedPrice = price
		m.cachedAt = time.Now()
		return price
	}

	if price := m.fetchBinanceTicker(); price > 0 {
		m.cachedPrice = price
		m.cachedAt = time.Now()
		return price
	}

	return m.cachedPrice
}

func (m *DemoLiquidityManager) fetchCoinGeckoSimple() int64 {
	baseURL := "https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd"
	urls := []string{baseURL}
	if m.apiKey != "" {
		urls = append([]string{baseURL + "&x_cg_demo_api_key=" + m.apiKey}, urls...)
	}

	for _, u := range urls {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			cancel()
			continue
		}
		if m.apiKey != "" {
			req.Header.Set("x-cg-demo-api-key", m.apiKey)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			continue
		}

		if resp.StatusCode >= 400 {
			resp.Body.Close()
			cancel()
			continue
		}

		var payload struct {
			Bitcoin struct {
				USD float64 `json:"usd"`
			} `json:"bitcoin"`
		}
		err = json.NewDecoder(resp.Body).Decode(&payload)
		resp.Body.Close()
		cancel()
		if err != nil {
			continue
		}

		price := engine.PriceToInt(payload.Bitcoin.USD)
		if price > 0 {
			return price
		}
	}

	return 0
}

func (m *DemoLiquidityManager) fetchBinanceTicker() int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT", nil)
	if err != nil {
		return 0
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0
	}

	var payload struct {
		Price string `json:"price"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0
	}

	p, err := strconv.ParseFloat(payload.Price, 64)
	if err != nil {
		return 0
	}

	return engine.PriceToInt(p)
}

func (m *DemoLiquidityManager) pulseFlow(server *ExchangeServer) {
	snapshot, err := server.StreamOrderBook(context.Background(), &pb.OrderBookRequest{Market: "demo"})
	if err != nil {
		return
	}

	sideBuy := !m.lastSideBuy
	if sideBuy && len(snapshot.Asks) == 0 {
		sideBuy = false
	}
	if !sideBuy && len(snapshot.Bids) == 0 {
		sideBuy = true
	}

	var qty int64
	if sideBuy {
		if len(snapshot.Asks) == 0 {
			return
		}
		qty = max(1, snapshot.Asks[0].TotalVolume/6)
	} else {
		if len(snapshot.Bids) == 0 {
			return
		}
		qty = max(1, snapshot.Bids[0].TotalVolume/6)
	}

	order := &pb.Order{
		Id:        time.Now().UnixNano(),
		Bid:       sideBuy,
		Size:      qty,
		Timestamp: time.Now().UnixNano(),
	}

	defer func() {
		_ = recover()
	}()

	_, _ = server.PlaceMarketOrder(context.Background(), order)
	m.lastSideBuy = sideBuy
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func (m *DemoLiquidityManager) pickMidPrice(snapshot *pb.OrderbookSnapshot) int64 {
	if snapshot == nil {
		return 0
	}
	if len(snapshot.Bids) > 0 && len(snapshot.Asks) > 0 {
		return (snapshot.Bids[0].Price + snapshot.Asks[0].Price) / 2
	}
	if len(snapshot.Bids) > 0 {
		return snapshot.Bids[0].Price
	}
	if len(snapshot.Asks) > 0 {
		return snapshot.Asks[0].Price
	}
	return 0
}

func (m *DemoLiquidityManager) nextSize() int64 {
	jitter := 1 - m.jitterPct + m.rng.Float64()*(2*m.jitterPct)
	size := m.baseSize * jitter
	if size <= 0 {
		size = m.baseSize
	}
	units := engine.QuantityToInt(size)
	if units <= 0 {
		return 1
	}
	return units
}

func (m *DemoLiquidityManager) seedOneTradeTick(snapshot *pb.OrderbookSnapshot) {
	price := m.pickMidPrice(snapshot)
	if price <= 0 {
		price = engine.PriceToInt(m.basePriceUSD)
	}
	BroadcastTrade(TradeTick{
		Price:     price,
		Size:      m.nextSize(),
		Side:      "buy",
		Timestamp: time.Now().UnixMilli(),
	})
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "TRUE", "yes", "on":
		return true
	case "0", "false", "FALSE", "no", "off":
		return false
	default:
		return fallback
	}
}
