package service

import (
	"context"
	"sync"

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

func (s *ExchangeServer) PlaceMarketOrder(ctx context.Context, order *pb.Order) (*pb.Match, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Wrap the protobuf order in the engine Order type
	engineOrder := &engine.Order{
		Order: order,
	}

	matches := s.orderbook.PlaceMarketOrder(engineOrder)

	go BroadcastOrderBook(s)

	// Return the first match if any, or an empty match
	if len(matches) > 0 {
		firstMatch := matches[0]
		return &pb.Match{
			Ask: &pb.Order{
				Id: firstMatch.Ask.Id, Price: firstMatch.Ask.Price,
				Size: firstMatch.Ask.Size, Bid: firstMatch.Ask.Bid,
				Timestamp: firstMatch.Ask.Timestamp,
			},
			Bid: &pb.Order{
				Id: firstMatch.Bid.Id, Price: firstMatch.Bid.Price,
				Size: firstMatch.Bid.Size, Bid: firstMatch.Bid.Bid,
				Timestamp: firstMatch.Bid.Timestamp,
			},
			SizeFilled: firstMatch.SizeFilled,
			Price:      firstMatch.Price,
		}, nil
	}

	// Return empty match if no matches occurred
	return &pb.Match{}, nil
}

func (s *ExchangeServer) PlaceLimitOrder(ctx context.Context, req *pb.PlaceLimitOrderRequest) (*pb.Match, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Wrap the protobuf order in the engine Order type
	engineOrder := &engine.Order{
		Order: req.Order,
	}

	s.orderbook.PlaceLimitOrder(req.Price, engineOrder)

	go BroadcastOrderBook(s)

	// Since PlaceLimitOrder doesn't return matches, return an empty match
	return &pb.Match{
		Ask: &pb.Order{
			Id: req.Order.Id, Price: req.Order.Price,
			Size: req.Order.Size, Bid: req.Order.Bid,
			Timestamp: req.Order.Timestamp,
		},
		Price: req.Price,
	}, nil
}

func (s *ExchangeServer) StreamOrderBook(ctx context.Context, req *pb.OrderBookRequest) (*pb.OrderbookSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := s.orderbook.GetSnapshot()
	return snapshot, nil
}
