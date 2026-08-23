package benchmarks

import (
	"time"

	"github.com/kayden-vs/zaraba/internal/engine"
	"github.com/kayden-vs/zaraba/internal/service"
	"github.com/kayden-vs/zaraba/pb"
)

// buildSnapshot returns an OrderbookSnapshot with depth ask and bid levels,
// each carrying a couple of resting orders to reflect real book structure.
func buildSnapshot(depth int) *pb.OrderbookSnapshot {
	s := &pb.OrderbookSnapshot{Timestamp: time.Now().UnixMilli()}
	basePrice := int64(45_000_000_000) // $45,000 in micro-USDT

	for i := 0; i < depth; i++ {
		askPrice := basePrice + int64(i+1)*1_000_000 // +$1 per level
		bidPrice := basePrice - int64(i+1)*1_000_000

		s.Asks = append(s.Asks, &pb.Limit{
			Price:       askPrice,
			TotalVolume: int64(100_000_000 * (depth - i)), // satoshis
			Orders: []*pb.Order{
				{Id: int64(i*2 + 1), Price: askPrice, Size: int64(50_000_000 * (depth - i)), Bid: false, Timestamp: time.Now().UnixNano()},
				{Id: int64(i*2 + 2), Price: askPrice, Size: int64(50_000_000 * (depth - i)), Bid: false, Timestamp: time.Now().UnixNano()},
			},
		})
		s.Bids = append(s.Bids, &pb.Limit{
			Price:       bidPrice,
			TotalVolume: int64(100_000_000 * (depth - i)),
			Orders: []*pb.Order{
				{Id: int64(depth*2+i*2+1), Price: bidPrice, Size: int64(50_000_000 * (depth - i)), Bid: true, Timestamp: time.Now().UnixNano()},
				{Id: int64(depth*2+i*2+2), Price: bidPrice, Size: int64(50_000_000 * (depth - i)), Bid: true, Timestamp: time.Now().UnixNano()},
			},
		})
	}
	return s
}

// buildExchangeServer returns an ExchangeServer pre-loaded with depth resting
// ask and bid limit orders, ready for market/limit order benchmarks.
func buildExchangeServer(depth int) *service.ExchangeServer {
	srv := service.NewExchangeServer()
	basePrice := int64(45_000_000_000)

	for i := 0; i < depth; i++ {
		askPrice := basePrice + int64(i+1)*1_000_000
		bidPrice := basePrice - int64(i+1)*1_000_000
		size := int64(10_000_000 * (depth - i + 1)) // small enough that orders rest

		// resting ask
		srv.PlaceLimitOrder(nil, &pb.PlaceLimitOrderRequest{
			Price: askPrice,
			Order: &pb.Order{Id: int64(i*2 + 1), Price: askPrice, Size: size, Bid: false, Timestamp: time.Now().UnixNano()},
		})
		// resting bid
		srv.PlaceLimitOrder(nil, &pb.PlaceLimitOrderRequest{
			Price: bidPrice,
			Order: &pb.Order{Id: int64(i*2 + 2), Price: bidPrice, Size: size, Bid: true, Timestamp: time.Now().UnixNano()},
		})
	}
	return srv
}

// restOrderRequest mirrors the JSON body a REST client would send for a limit order.
type restOrderRequest struct {
	Price     int64 `json:"price"`
	Size      int64 `json:"size"`
	Bid       bool  `json:"bid"`
	ID        int64 `json:"id"`
	Timestamp int64 `json:"timestamp"`
}

// restMatchResponse mirrors the JSON body a REST server would return.
type restMatchResponse struct {
	Ask        *restOrderRequest `json:"ask,omitempty"`
	Bid        *restOrderRequest `json:"bid,omitempty"`
	SizeFilled int64             `json:"size_filled"`
	Price      int64             `json:"price"`
}

// restLimitLevel mirrors a single price level in a REST /orderbook response.
type restLimitLevel struct {
	Price       int64 `json:"price"`
	TotalVolume int64 `json:"total_volume"`
}

// restSnapshot mirrors what a REST /orderbook endpoint would return.
type restSnapshot struct {
	Asks      []restLimitLevel `json:"asks"`
	Bids      []restLimitLevel `json:"bids"`
	Timestamp int64            `json:"timestamp"`
}

// idSeed drives unique order IDs across benchmark iterations.
var idSeed = time.Now().UnixNano()

func nextID() int64 {
	idSeed++
	return idSeed
}

// matchToRest converts pb.Match to restMatchResponse (simulates REST serialisation path).
func matchToRest(m *pb.Match) restMatchResponse {
	r := restMatchResponse{SizeFilled: m.SizeFilled, Price: m.Price}
	if m.Ask != nil {
		r.Ask = &restOrderRequest{ID: m.Ask.Id, Price: m.Ask.Price, Size: m.Ask.Size, Bid: m.Ask.Bid}
	}
	if m.Bid != nil {
		r.Bid = &restOrderRequest{ID: m.Bid.Id, Price: m.Bid.Price, Size: m.Bid.Size, Bid: m.Bid.Bid}
	}
	return r
}

// snapshotToRest flattens a pb.OrderbookSnapshot into a REST-style payload
// (drops per-order detail to match a typical REST API response).
func snapshotToRest(s *pb.OrderbookSnapshot) restSnapshot {
	r := restSnapshot{Timestamp: s.Timestamp}
	for _, l := range s.Asks {
		r.Asks = append(r.Asks, restLimitLevel{Price: l.Price, TotalVolume: l.TotalVolume})
	}
	for _, l := range s.Bids {
		r.Bids = append(r.Bids, restLimitLevel{Price: l.Price, TotalVolume: l.TotalVolume})
	}
	return r
}

// buildEngineOrderbook seeds a plain engine.Orderbook with depth levels —
// used by benchmarks that need to bypass the gRPC service layer entirely.
func buildEngineOrderbook(depth int) *engine.Orderbook {
	ob := engine.NewOrderbook()
	basePrice := int64(45_000_000_000)

	for i := 0; i < depth; i++ {
		askPrice := basePrice + int64(i+1)*1_000_000
		bidPrice := basePrice - int64(i+1)*1_000_000
		size := int64(10_000_000 * (depth - i + 1))

		ob.PlaceLimitOrder(askPrice, &engine.Order{Order: &pb.Order{Id: int64(i*2 + 1), Price: askPrice, Size: size, Bid: false, Timestamp: time.Now().UnixNano()}})
		ob.PlaceLimitOrder(bidPrice, &engine.Order{Order: &pb.Order{Id: int64(i*2 + 2), Price: bidPrice, Size: size, Bid: true, Timestamp: time.Now().UnixNano()}})
	}
	return ob
}
