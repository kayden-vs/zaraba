package main

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

	// Place the market order and get matches
	matches := s.orderbook.PlaceMarketOrder(engineOrder)

	// Return the first match if any, or an empty match
	if len(matches) > 0 {
		firstMatch := matches[0]
		return &pb.Match{
			Ask:        firstMatch.Ask.Order,
			Bid:        firstMatch.Bid.Order,
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

	// Place the limit order
	s.orderbook.PlaceLimitOrder(req.Price, engineOrder)

	// Since PlaceLimitOrder doesn't return matches, return an empty match
	return &pb.Match{
		Ask:   req.Order,
		Price: req.Price,
	}, nil
}
