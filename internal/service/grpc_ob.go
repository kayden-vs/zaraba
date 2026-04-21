package service

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/kayden-vs/zaraba/internal/engine"
	"github.com/kayden-vs/zaraba/pb"
)

// ExchangeServer implements the gRPC Exchange service
type ExchangeServer struct {
	pb.UnimplementedExchangeServer
	orderbook *engine.Orderbook
	mu        sync.RWMutex
}

// NewExchangeServer creates a new Exchange gRPC server
func NewExchangeServer() *ExchangeServer {
	return &ExchangeServer{
		orderbook: engine.NewOrderbook(),
	}
}

func (s *ExchangeServer) ResetOrderbook() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orderbook = engine.NewOrderbook()
	go BroadcastOrderBook(s)
}

func (s *ExchangeServer) PlaceMarketOrder(ctx context.Context, order *pb.Order) (*pb.Match, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Wrap the protobuf order in the engine Order type
	engineOrder := &engine.Order{
		Order: order,
	}

	matches := s.orderbook.PlaceMarketOrder(engineOrder)

	go BroadcastOrderBook(s)

	takerSide := "sell"
	if order.Bid {
		takerSide = "buy"
	}
	for _, m := range matches {
		if m.SizeFilled <= 0 {
			continue
		}
		BroadcastTrade(TradeTick{
			Price:     m.Price,
			Size:      m.SizeFilled,
			Side:      takerSide,
			Timestamp: time.Now().UnixMilli(),
		})
	}

	return aggregateMatch(matches), nil
}

func (s *ExchangeServer) PlaceLimitOrder(ctx context.Context, req *pb.PlaceLimitOrderRequest) (*pb.Match, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Wrap the protobuf order in the engine Order type
	engineOrder := &engine.Order{
		Order: req.Order,
	}

	matches := s.orderbook.PlaceLimitOrder(req.Price, engineOrder)

	go BroadcastOrderBook(s)

	takerSide := "sell"
	if req.Order.Bid {
		takerSide = "buy"
	}
	for _, m := range matches {
		if m.SizeFilled <= 0 {
			continue
		}
		BroadcastTrade(TradeTick{
			Price:     m.Price,
			Size:      m.SizeFilled,
			Side:      takerSide,
			Timestamp: time.Now().UnixMilli(),
		})
	}

	return aggregateMatch(matches), nil
}

func aggregateMatch(matches []engine.Match) *pb.Match {
	if len(matches) == 0 {
		return &pb.Match{}
	}

	first := matches[0]
	var totalFilled int64
	var totalNotional float64
	for _, m := range matches {
		if m.SizeFilled <= 0 || m.Price <= 0 {
			continue
		}
		totalFilled += m.SizeFilled
		totalNotional += float64(m.Price) * float64(m.SizeFilled)
	}

	vwap := first.Price
	if totalFilled > 0 {
		vwap = int64(math.Round(totalNotional / float64(totalFilled)))
	}

	return &pb.Match{
		Ask: &pb.Order{
			Id: first.Ask.Id, Price: first.Ask.Price,
			Size: first.Ask.Size, Bid: first.Ask.Bid,
			Timestamp: first.Ask.Timestamp,
		},
		Bid: &pb.Order{
			Id: first.Bid.Id, Price: first.Bid.Price,
			Size: first.Bid.Size, Bid: first.Bid.Bid,
			Timestamp: first.Bid.Timestamp,
		},
		SizeFilled: totalFilled,
		Price:      vwap,
	}
}

func (s *ExchangeServer) StreamOrderBook(ctx context.Context, req *pb.OrderBookRequest) (*pb.OrderbookSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := s.orderbook.GetSnapshot()
	return snapshot, nil
}

func (s *ExchangeServer) CancelOrderByID(orderID int64) (*pb.Order, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cancelled, ok := s.orderbook.CancelOrderByID(orderID)
	if ok {
		go BroadcastOrderBook(s)
	}

	return cancelled, ok
}
