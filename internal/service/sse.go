package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/kayden-vs/zaraba/pb"
	"github.com/kayden-vs/zaraba/ui/html/pages"
)

// For Server Side Events
// To update markets page and Orderbook

type Broker struct {
	clients map[chan []byte]bool
	mu      sync.Mutex
}

const (
	MaxOrderBookLevels = 20
	MaxRecentTrades    = 100
)

type TradeTick struct {
	Price     int64  `json:"price"`
	Size      int64  `json:"size"`
	Side      string `json:"side"`
	Timestamp int64  `json:"timestamp"`
}

var PriceBroker = &Broker{
	clients: make(map[chan []byte]bool),
}

var OrderbookBroker = &Broker{
	clients: make(map[chan []byte]bool),
}

var TradeBroker = &Broker{
	clients: make(map[chan []byte]bool),
}

var tradeStore = struct {
	mu     sync.RWMutex
	trades []TradeTick
}{
	trades: make([]TradeTick, 0, MaxRecentTrades),
}

func (b *Broker) AddClient() chan []byte {
	ch := make(chan []byte, 5) // buffered so slow clients dont block
	b.mu.Lock()
	b.clients[ch] = true
	b.mu.Unlock()
	return ch
}

func (b *Broker) RemoveClient(ch chan []byte) {
	b.mu.Lock()
	delete(b.clients, ch)
	close(ch)
	b.mu.Unlock()
}

func (b *Broker) Broadcast(data []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for ch := range b.clients {
		select {
		case ch <- data: // send to client
		default: // client is slow/blocked, skip them
			log.Println("skipping slow client")
		}
	}
}

func (b *Broker) ClientCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.clients)
}

func StartMarketFetcher() {
	url := "https://api.coingecko.com/api/v3/coins/markets" +
		"?vs_currency=usd" +
		"&order=market_cap_desc" +
		"&per_page=10" +
		"&page=1" +
		"&price_change_percentage=24h" +
		fmt.Sprintf("&x_cg_demo_api_key=%s", os.Getenv("API_KEY"))

	for {
		if PriceBroker.ClientCount() == 0 {
			time.Sleep(5 * time.Second)
			// log.Println("No client, skipping fetch")
			continue
		}

		resp, err := http.Get(url)
		if err != nil {
			log.Println("Error fetching price:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var symbolData []pages.CoinMarketProps
		if err := json.NewDecoder(resp.Body).Decode(&symbolData); err != nil {
			log.Println("Error decoding response:", err)
			resp.Body.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		resp.Body.Close()

		jsonData, err := json.Marshal(symbolData)
		if err != nil {
			log.Println("Error marshaling data:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		PriceBroker.Broadcast(jsonData) // sends to all clients

		time.Sleep(5 * time.Second)
	}
}

// direct broadcast on order placement
func BroadcastOrderBook(server *ExchangeServer) {
	snapshot, err := server.StreamOrderBook(context.Background(), &pb.OrderBookRequest{Market: "btc"})
	if err != nil {
		log.Println("Error fetching orderbook:", err)
		return
	}

	snapshot = trimSnapshot(snapshot, MaxOrderBookLevels)

	jsonData, err := json.Marshal(snapshot)
	if err != nil {
		log.Println("Error marshaling snapshot:", err)
		return
	}

	OrderbookBroker.Broadcast(jsonData)
}

func BroadcastTrade(tick TradeTick) {
	tradeStore.mu.Lock()
	tradeStore.trades = append([]TradeTick{tick}, tradeStore.trades...)
	if len(tradeStore.trades) > MaxRecentTrades {
		tradeStore.trades = tradeStore.trades[:MaxRecentTrades]
	}
	tradeStore.mu.Unlock()

	jsonData, err := json.Marshal(tick)
	if err != nil {
		log.Println("Error marshaling trade:", err)
		return
	}

	TradeBroker.Broadcast(jsonData)
}

func RecentTradesPayload() []byte {
	tradeStore.mu.RLock()
	defer tradeStore.mu.RUnlock()

	jsonData, err := json.Marshal(tradeStore.trades)
	if err != nil {
		return []byte("[]")
	}

	return jsonData
}

func trimSnapshot(snapshot *pb.OrderbookSnapshot, depth int) *pb.OrderbookSnapshot {
	if snapshot == nil || depth <= 0 {
		return snapshot
	}

	trimmed := &pb.OrderbookSnapshot{Timestamp: snapshot.Timestamp}

	askDepth := min(depth, len(snapshot.Asks))
	bidDepth := min(depth, len(snapshot.Bids))

	trimmed.Asks = snapshot.Asks[:askDepth]
	trimmed.Bids = snapshot.Bids[:bidDepth]

	return trimmed
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
