package benchmarks

// Benchmark 2: SSE + channel hub vs WebSocket hub for live orderbook/price broadcasting.
//
// The current SSE Broker fans out []byte to per-client buffered channels;
// the HTTP handler reads from its channel and writes "data: ...\n\n" frames.
// A WebSocket alternative requires a goroutine per connection and uses
// nhooyr.io/websocket WriteMessage — which holds a write lock per send.
//
// Key differences being measured:
//   - Fan-out latency to N concurrent clients
//   - Memory footprint per connected client
//   - Connection setup/teardown overhead

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/kayden-vs/zaraba/internal/service"
	nhws "nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// ── SSE broker helpers ───────────────────────────────────────────────────────

// sseSimClient simulates a connected SSE client: a buffered channel that
// drains incoming data, matching what SseOrderBookHandler does in production.
type sseSimClient struct {
	ch   chan []byte
	done chan struct{}
}

func newSSESimClient(broker *service.Broker) *sseSimClient {
	c := &sseSimClient{
		ch:   broker.AddClient(),
		done: make(chan struct{}),
	}
	go func() {
		for range c.ch { // drain so the broker never blocks
		}
		close(c.done)
	}()
	return c
}

func (c *sseSimClient) close(broker *service.Broker) {
	broker.RemoveClient(c.ch)
	<-c.done
}

// ── WebSocket hub (benchmark alternative) ────────────────────────────────────

// wsHub mirrors the SSE Broker interface but uses WebSocket connections.
type wsHub struct {
	mu      sync.Mutex
	clients []*nhws.Conn
}

func (h *wsHub) addClient(conn *nhws.Conn) {
	h.mu.Lock()
	h.clients = append(h.clients, conn)
	h.mu.Unlock()
}

func (h *wsHub) broadcast(ctx context.Context, v any) {
	h.mu.Lock()
	conns := make([]*nhws.Conn, len(h.clients))
	copy(conns, h.clients)
	h.mu.Unlock()

	for _, conn := range conns {
		_ = wsjson.Write(ctx, conn, v) // ignores closed connections
	}
}

func (h *wsHub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, conn := range h.clients {
		_ = conn.CloseNow()
	}
	h.clients = nil
}

// newWsEchoServer returns an HTTP handler that upgrades to WebSocket and
// drains all received messages — standard read-pump server pattern.
func newWsEchoServer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := nhws.Accept(w, r, &nhws.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer conn.CloseNow()
		for {
			if _, _, err := conn.Read(r.Context()); err != nil {
				return
			}
		}
	})
}

// ── Fan-out benchmarks ───────────────────────────────────────────────────────

func BenchmarkSSE_Broadcast_10Clients(b *testing.B)  { benchmarkSSEBroadcast(b, 10) }
func BenchmarkSSE_Broadcast_100Clients(b *testing.B) { benchmarkSSEBroadcast(b, 100) }
func BenchmarkSSE_Broadcast_500Clients(b *testing.B) { benchmarkSSEBroadcast(b, 500) }
func BenchmarkSSE_Broadcast_1KClients(b *testing.B)  { benchmarkSSEBroadcast(b, 1_000) }

func BenchmarkWS_Broadcast_10Clients(b *testing.B)  { benchmarkWSBroadcast(b, 10) }
func BenchmarkWS_Broadcast_100Clients(b *testing.B) { benchmarkWSBroadcast(b, 100) }
func BenchmarkWS_Broadcast_500Clients(b *testing.B) { benchmarkWSBroadcast(b, 500) }
func BenchmarkWS_Broadcast_1KClients(b *testing.B)  { benchmarkWSBroadcast(b, 1_000) }

// benchmarkSSEBroadcast measures how long the Broker takes to push one
// orderbook payload to n simultaneous client channels.
func benchmarkSSEBroadcast(b *testing.B, n int) {
	broker := service.NewBroker()

	clients := make([]*sseSimClient, n)
	for i := range clients {
		clients[i] = newSSESimClient(broker)
	}
	defer func() {
		for _, c := range clients {
			c.close(broker)
		}
	}()

	payload, _ := json.Marshal(buildSnapshot(20))

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))

	for i := 0; i < b.N; i++ {
		broker.Broadcast(payload)
	}
}

// benchmarkWSBroadcast measures fan-out to n WebSocket connections.
// Each client is a real (loopback) WebSocket connection so the measurement
// includes write-lock acquisition and framing overhead.
func benchmarkWSBroadcast(b *testing.B, n int) {
	hub := &wsHub{}
	server := httptest.NewServer(newWsEchoServer())
	b.Cleanup(server.Close)

	for i := 0; i < n; i++ {
		conn, _, err := nhws.Dial(context.Background(), "ws"+server.URL[len("http"):], nil)
		if err != nil {
			b.Fatalf("dial client %d: %v", i, err)
		}
		hub.addClient(conn)
		// drain in background so server writes don't block
		go func(c *nhws.Conn) {
			for {
				if _, _, err := c.Read(context.Background()); err != nil {
					return
				}
			}
		}(conn)
	}
	defer hub.closeAll()

	payload := buildSnapshot(20)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		hub.broadcast(context.Background(), payload)
	}
}

// ── Connection setup benchmarks ───────────────────────────────────────────────

// BenchmarkSSE_ConnectionSetup measures the cost of registering a new SSE client:
// channel allocation + map insertion into the Broker.
func BenchmarkSSE_ConnectionSetup(b *testing.B) {
	broker := service.NewBroker()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ch := broker.AddClient()
		broker.RemoveClient(ch)
	}
}

// BenchmarkWS_ConnectionSetup measures the cost of a full WebSocket upgrade
// (TCP dial + HTTP upgrade handshake) over loopback.
func BenchmarkWS_ConnectionSetup(b *testing.B) {
	server := httptest.NewServer(newWsEchoServer())
	b.Cleanup(server.Close)
	wsURL := "ws" + server.URL[len("http"):]

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		conn, _, err := nhws.Dial(context.Background(), wsURL, nil)
		if err != nil {
			b.Fatalf("dial: %v", err)
		}
		_ = conn.CloseNow()
	}
}

// ── Memory-per-client benchmarks ──────────────────────────────────────────────

// BenchmarkSSE_MemoryPerClient reports heap allocated per SSE client handle.
func BenchmarkSSE_MemoryPerClient(b *testing.B) {
	broker := service.NewBroker()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ch := broker.AddClient()
		_ = ch
		broker.RemoveClient(ch)
	}
}

// BenchmarkWS_MemoryPerClient reports heap allocated per WebSocket connection
// (upgrade + internal buffers allocated by nhooyr.io/websocket).
func BenchmarkWS_MemoryPerClient(b *testing.B) {
	server := httptest.NewServer(newWsEchoServer())
	b.Cleanup(server.Close)
	wsURL := "ws" + server.URL[len("http"):]

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		conn, _, err := nhws.Dial(context.Background(), wsURL, nil)
		if err != nil {
			b.Fatalf("dial: %v", err)
		}
		_ = conn.CloseNow()
	}
}
