package main

import (
	"context"
	"log"

	"github.com/kayden-vs/zaraba/internal/engine"
	"github.com/kayden-vs/zaraba/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewExchangeClient(conn)
	ctx := context.Background()

	// Place some limit buy orders (bids)
	bids := [][2]float64{
		{96000, 0.5},
		{95500, 1.2},
		{95000, 0.8},
	}
	for _, b := range bids {
		client.PlaceLimitOrder(ctx, &pb.PlaceLimitOrderRequest{
			Price: engine.PriceToInt(b[0]),
			Order: &pb.Order{Bid: true, Size: engine.QuantityToInt(b[1])},
		})
	}

	// Place some limit sell orders (asks)
	asks := [][2]float64{
		{96500, 0.3},
		{97000, 0.9},
		{97500, 0.5},
		{98000, 1.0},
	}
	for _, a := range asks {
		client.PlaceLimitOrder(ctx, &pb.PlaceLimitOrderRequest{
			Price: engine.PriceToInt(a[0]),
			Order: &pb.Order{Bid: false, Size: engine.QuantityToInt(a[1])},
		})
	}

	log.Println("Seeded orderbook successfully")
}
