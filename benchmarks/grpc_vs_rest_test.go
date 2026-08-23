package benchmarks

// Benchmark 1: gRPC (direct method call with Protobuf types) vs
// JSON/REST (marshal request -> call handler -> unmarshal response).
//
// The current architecture calls ExchangeServer methods directly inside
// HTTP handlers — no network hop, binary Protobuf types in memory.
// A REST alternative would require JSON encoding on both sides of
// every order placement, adding serialisation overhead to the hot path.

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kayden-vs/zaraba/pb"
)

// ── Limit order: place and rest ──────────────────────────────────────────────

func BenchmarkGRPC_PlaceLimitOrder_Depth100(b *testing.B) {
	benchmarkGRPCPlaceLimitOrder(b, 100)
}
func BenchmarkGRPC_PlaceLimitOrder_Depth1K(b *testing.B) {
	benchmarkGRPCPlaceLimitOrder(b, 1_000)
}
func BenchmarkGRPC_PlaceLimitOrder_Depth10K(b *testing.B) {
	benchmarkGRPCPlaceLimitOrder(b, 10_000)
}

func BenchmarkREST_PlaceLimitOrder_Depth100(b *testing.B) {
	benchmarkRESTPlaceLimitOrder(b, 100)
}
func BenchmarkREST_PlaceLimitOrder_Depth1K(b *testing.B) {
	benchmarkRESTPlaceLimitOrder(b, 1_000)
}
func BenchmarkREST_PlaceLimitOrder_Depth10K(b *testing.B) {
	benchmarkRESTPlaceLimitOrder(b, 10_000)
}

// benchmarkGRPCPlaceLimitOrder measures the direct Protobuf call path:
// the caller already holds typed structs; no serialisation needed.
func benchmarkGRPCPlaceLimitOrder(b *testing.B, depth int) {
	srv := buildExchangeServer(depth)
	basePrice := int64(45_000_000_000)
	newAskPrice := basePrice + int64(depth+1)*1_000_000 // fresh level that won't cross

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = srv.PlaceLimitOrder(context.Background(), &pb.PlaceLimitOrderRequest{
			Price: newAskPrice,
			Order: &pb.Order{
				Id:        nextID(),
				Price:     newAskPrice,
				Size:      1_000_000,
				Bid:       false,
				Timestamp: time.Now().UnixNano(),
			},
		})
	}
}

// benchmarkRESTPlaceLimitOrder simulates the REST alternative path:
// JSON-encode the request, call the handler, JSON-encode the response.
// This mirrors what a net/http handler would do with a JSON body.
func benchmarkRESTPlaceLimitOrder(b *testing.B, depth int) {
	srv := buildExchangeServer(depth)
	basePrice := int64(45_000_000_000)
	newAskPrice := basePrice + int64(depth+1)*1_000_000

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate REST request: marshal JSON body
		req := restOrderRequest{
			ID:        nextID(),
			Price:     newAskPrice,
			Size:      1_000_000,
			Bid:       false,
			Timestamp: time.Now().UnixNano(),
		}
		reqBytes, _ := json.Marshal(req)

		// Simulate REST handler: unmarshal JSON body
		var decoded restOrderRequest
		_ = json.Unmarshal(reqBytes, &decoded)

		// Actual orderbook call (same work as gRPC path)
		match, _ := srv.PlaceLimitOrder(context.Background(), &pb.PlaceLimitOrderRequest{
			Price: decoded.Price,
			Order: &pb.Order{
				Id:        decoded.ID,
				Price:     decoded.Price,
				Size:      decoded.Size,
				Bid:       decoded.Bid,
				Timestamp: decoded.Timestamp,
			},
		})

		// Simulate REST response: marshal JSON body
		resp := matchToRest(match)
		_, _ = json.Marshal(resp)
	}
}

// ── Market order ─────────────────────────────────────────────────────────────

func BenchmarkGRPC_PlaceMarketOrder_Depth100(b *testing.B) {
	benchmarkGRPCPlaceMarketOrder(b, 100)
}
func BenchmarkGRPC_PlaceMarketOrder_Depth1K(b *testing.B) {
	benchmarkGRPCPlaceMarketOrder(b, 1_000)
}

func BenchmarkREST_PlaceMarketOrder_Depth100(b *testing.B) {
	benchmarkRESTPlaceMarketOrder(b, 100)
}
func BenchmarkREST_PlaceMarketOrder_Depth1K(b *testing.B) {
	benchmarkRESTPlaceMarketOrder(b, 1_000)
}

// benchmarkGRPCPlaceMarketOrder measures a market buy that hits the best ask —
// minimal depth traversal, so the difference is purely serialisation overhead.
func benchmarkGRPCPlaceMarketOrder(b *testing.B, depth int) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		srv := buildExchangeServer(depth) // fresh book so order always fills
		b.StartTimer()

		_, _ = srv.PlaceMarketOrder(context.Background(), &pb.Order{
			Id:        nextID(),
			Bid:       true,
			Size:      1_000, // tiny — guaranteed to fill against first ask level
			Timestamp: time.Now().UnixNano(),
		})
	}
}

// benchmarkRESTPlaceMarketOrder adds JSON round-trip overhead to the market order path.
func benchmarkRESTPlaceMarketOrder(b *testing.B, depth int) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		srv := buildExchangeServer(depth)
		b.StartTimer()

		// Simulate REST request marshal/unmarshal
		req := restOrderRequest{
			ID:        nextID(),
			Bid:       true,
			Size:      1_000,
			Timestamp: time.Now().UnixNano(),
		}
		reqBytes, _ := json.Marshal(req)
		var decoded restOrderRequest
		_ = json.Unmarshal(reqBytes, &decoded)

		match, _ := srv.PlaceMarketOrder(context.Background(), &pb.Order{
			Id:        decoded.ID,
			Bid:       decoded.Bid,
			Size:      decoded.Size,
			Timestamp: decoded.Timestamp,
		})

		// Simulate REST response marshal
		resp := matchToRest(match)
		_, _ = json.Marshal(resp)
	}
}

// ── Orderbook snapshot (GET /orderbook equivalent) ───────────────────────────

func BenchmarkGRPC_GetSnapshot_Depth20(b *testing.B)  { benchmarkGRPCGetSnapshot(b, 20) }
func BenchmarkGRPC_GetSnapshot_Depth100(b *testing.B) { benchmarkGRPCGetSnapshot(b, 100) }
func BenchmarkGRPC_GetSnapshot_Depth500(b *testing.B) { benchmarkGRPCGetSnapshot(b, 500) }

func BenchmarkREST_GetSnapshot_Depth20(b *testing.B)  { benchmarkRESTGetSnapshot(b, 20) }
func BenchmarkREST_GetSnapshot_Depth100(b *testing.B) { benchmarkRESTGetSnapshot(b, 100) }
func BenchmarkREST_GetSnapshot_Depth500(b *testing.B) { benchmarkRESTGetSnapshot(b, 500) }

// benchmarkGRPCGetSnapshot retrieves an orderbook snapshot and gets back
// typed Protobuf structs — the caller can use fields directly with no parsing.
func benchmarkGRPCGetSnapshot(b *testing.B, depth int) {
	srv := buildExchangeServer(depth)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = srv.StreamOrderBook(context.Background(), &pb.OrderBookRequest{Market: "btc"})
	}
}

// benchmarkRESTGetSnapshot retrieves a snapshot and JSON-encodes it for the
// HTTP response body — what a REST /orderbook handler must do on every poll.
func benchmarkRESTGetSnapshot(b *testing.B, depth int) {
	srv := buildExchangeServer(depth)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		snapshot, _ := srv.StreamOrderBook(context.Background(), &pb.OrderBookRequest{Market: "btc"})
		payload := snapshotToRest(snapshot)
		_, _ = json.Marshal(payload)
	}
}
