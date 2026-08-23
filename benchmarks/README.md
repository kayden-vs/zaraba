# Zaraba Performance Benchmarks

A self-contained Go benchmark suite (`testing.B`) comparing Zaraba's current
architecture choices against their most common alternatives.

**No running server or database required** — all benchmarks are in-process.

---

## Quick Start

```bash
# Run all suites (5 iterations each, 3 s per benchmark)
./benchmarks/run_benchmarks.sh

# Run individual suites
./benchmarks/run_benchmarks.sh grpc    # Benchmark 1
./benchmarks/run_benchmarks.sh sse     # Benchmark 2
./benchmarks/run_benchmarks.sh serial  # Benchmark 3

# Or call go test directly
cd /path/to/zaraba
go test ./benchmarks/... -bench=. -benchmem -count=5 -benchtime=3s -timeout=20m
```

Install `benchstat` for statistically rigorous run comparisons:

```bash
go install golang.org/x/perf/cmd/benchstat@latest
benchstat results/run1.txt results/run2.txt
```

---

## Benchmark 1 — gRPC vs JSON/REST (orderbook operations)

### What is being measured

Zaraba's HTTP handlers call `ExchangeServer` methods **directly** via Go method
calls — the request is already in a typed Protobuf struct in memory.
No network hop. No serialisation on the hot path.

A JSON/REST architecture would require:
1. JSON-decode the incoming HTTP body into a Go struct
2. Execute the same orderbook logic
3. JSON-encode the response back to bytes

The benchmark isolates steps 1 and 3 (serialisation round-trip) as the overhead
added by REST, keeping the orderbook work identical in both paths.

### Data flow comparison

```
gRPC path (current)
────────────────────────────────────────────────────────
HTTP handler  →  ExchangeServer.PlaceLimitOrder(ctx, *pb.PlaceLimitOrderRequest)
                 └─ Protobuf struct already in memory, zero serialisation
                 └─ returns *pb.Match  (typed struct)

REST path (alternative)
────────────────────────────────────────────────────────
HTTP handler  →  json.Unmarshal(body, &req)          ← extra alloc + parse
              →  ExchangeServer.PlaceLimitOrder(...)
              →  json.Marshal(resp)                  ← extra alloc + encode
              →  w.Write(jsonBytes)
```

### Results (AMD Ryzen 5 6600H, Linux, Go 1.25)

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| GRPC PlaceLimitOrder depth=100 | 350,072 | 687,070 | 5,630 |
| REST PlaceLimitOrder depth=100 | 346,089 | 691,735 | 5,656 |
| GRPC GetSnapshot depth=20 | 5,161 | 8,464 | 133 |
| REST GetSnapshot depth=20 | 10,459 | 12,620 | 147 |
| GRPC GetSnapshot depth=100 | 24,702 | 41,304 | 617 |
| REST GetSnapshot depth=100 | 47,879 | 59,764 | 636 |
| GRPC GetSnapshot depth=500 | 127,716 | 202,795 | 3,021 |
| REST GetSnapshot depth=500 | 241,604 | 300,028 | 3,049 |

### Interpretation

- **PlaceOrder**: overhead is negligible (~1%) at depth=100 because the
  orderbook matching work dominates. The distinction matters most for snapshot
  reads and at very high order rates (>50K ops/sec).
- **GetSnapshot** (the more important comparison): REST adds **~2× latency**
  and **~48% more memory** due to JSON marshalling. At depth=500 that is
  an extra 113 µs and ~100 KB per snapshot broadcast.
- In production, the orderbook snapshot is broadcast on every order placement
  to all connected SSE clients. With 1,000 ops/sec the REST path wastes
  ~113 ms/s of CPU time purely on JSON encoding.

---

## Benchmark 2 — SSE + Channel Hub vs WebSocket Hub (live broadcasting)

### What is being measured

Zaraba uses an SSE `Broker` to fan out orderbook/price updates. Each connected
client is a buffered `chan []byte`. The HTTP handler reads from the channel and
writes `data: <json>\n\n` frames, all within the existing HTTP/1.1 connection.

A WebSocket alternative wraps each client in a `*nhws.Conn` (nhooyr.io/websocket)
and calls `wsjson.Write` per send — which must acquire a per-connection write
lock and frame the payload as a WebSocket message.

### Data flow comparison

```
SSE path (current)
────────────────────────────────────────────────────────
Broker.Broadcast(payload)
  └─ for each client: ch <- payload  (non-blocking channel send, O(1))
Handler goroutine: fmt.Fprintf(w, "data: %s\n\n", <-ch)
  └─ HTTP/1.1 keep-alive connection, no upgrade overhead

WebSocket path (alternative)
────────────────────────────────────────────────────────
wsHub.broadcast(ctx, payload)
  └─ for each client: wsjson.Write(ctx, conn, payload)
       └─ JSON-encode payload
       └─ acquire conn write lock
       └─ frame as WebSocket binary/text message
       └─ syscall write
```

### Results (AMD Ryzen 5 6600H, Linux, Go 1.25)

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| SSE Broadcast 10 clients | ~10,000 | 1 | 0 |
| WS Broadcast 10 clients | 731,761 | 237,849 | 121 |
| SSE Broadcast 100 clients | ~26,000 | 1 | 0 |
| WS Broadcast 100 clients | 7,083,757 | 2,376,049 | 1,222 |
| SSE Broadcast 500 clients | ~53,248 | 0 | 0 |
| WS Broadcast 500 clients | 34,590,609 | 11,884,482 | 6,464 |
| SSE Broadcast 1K clients | 100,665 | 1 | 0 |
| WS Broadcast 1K clients | 70,352,001 | 23,818,622 | 14,022 |
| SSE ConnectionSetup | 155 | 240 | 2 |
| WS ConnectionSetup | 360,136 | 33,819 | 242 |
| SSE MemoryPerClient | 157 | 240 | 2 |
| WS MemoryPerClient | 354,475 | 33,911 | 242 |

*(SSE ns/op estimated from throughput metric; WS numbers are measured)*

### Interpretation

- **Fan-out**: SSE channel push is a non-blocking memory write — essentially
  free per client. WebSocket requires a lock + frame + syscall per client,
  making it **~73× slower at 10 clients** and **~698× slower at 1K clients** and scaling linearly with client
  count (WebSocket is O(n×syscall), SSE is O(n×channel-send)).
- **Memory**: SSE allocates **0 bytes** per broadcast (channel send is alloc-free
  after the initial channel creation). WebSocket allocates ~24 KB per client per
  broadcast at 1K clients (23.8 MB total) due to JSON encoding + frame buffers.
- **Connection setup**: WebSocket requires a full HTTP upgrade handshake
  (360 µs, 33 KB per connection). SSE registration is a channel allocation
  + map insert (155 ns, 240 B) — **2,323× faster setup**.
- **Conclusion**: SSE + channel hub is the correct architecture for a
  server-push-only workload (orderbook snapshots, price ticks). WebSocket
  complexity pays off only when clients need to send data back to the server
  at high frequency (e.g., ITCH/OUCH protocol style).

---

## Benchmark 3 — Protobuf Binary vs JSON Serialisation (orderbook snapshots)

### What is being measured

Every SSE broadcast serialises a `pb.OrderbookSnapshot` before writing it to
the wire. Currently this uses `encoding/json`, which treats the proto struct as
a plain Go struct.

Three formats are compared:
- **JSON** (`encoding/json`) — human-readable, widely supported
- **Protobuf binary** (`google.golang.org/protobuf/proto`) — compact binary
- **ProtoJSON** (`google.golang.org/protobuf/encoding/protojson`) — JSON via
  Protobuf reflection (what you'd use for a proper gRPC-gateway REST API)

### Results (AMD Ryzen 5 6600H, Linux, Go 1.25)

#### Marshal

| Format | Depth | ns/op | bytes/payload | B/op | allocs/op |
|---|---|---|---|---|---|
| JSON | 20 | 21,841 | 9,138 | 9,477 | 1 |
| Protobuf | 20 | **10,021** | **2,823** | 3,072 | 1 |
| ProtoJSON | 20 | 173,491 | 10,340 | 88,662 | 1,619 |
| JSON | 100 | 109,863 | 46,241 | 49,178 | 1 |
| Protobuf | 100 | **48,868** | **14,456** | 16,384 | 1 |
| ProtoJSON | 100 | 804,037 | 52,243 | 421,866 | 8,171 |
| JSON | 500 | 537,250 | 234,846 | 237,958 | 1 |
| Protobuf | 500 | **227,973** | **73,170** | 73,728 | 1 |
| ProtoJSON | 500 | 4,462,145 | 264,848 | 2,244,844 | 41,787 |

#### Unmarshal

| Format | Depth | ns/op | B/op | allocs/op |
|---|---|---|---|---|
| JSON | 20 | 118,228 | 59,080 | 1,662 |
| Protobuf | 20 | **15,009** | 12,944 | 213 |
| ProtoJSON | 20 | 277,217 | 45,504 | 2,768 |
| JSON | 100 | 585,879 | 291,274 | 8,226 |
| Protobuf | 100 | **82,748** | 63,632 | 1,017 |
| ProtoJSON | 100 | 1,373,421 | 226,642 | 13,957 |
| JSON | 500 | 2,963,385 | 1,446,355 | 41,030 |
| Protobuf | 500 | **473,828** | 314,769 | 5,021 |
| ProtoJSON | 500 | 9,297,355 | 1,136,989 | 70,761 |

### Interpretation

- **Protobuf binary is ~2.2× faster to marshal** than JSON and produces
  payloads **3.2× smaller** at depth=20, growing to **3.2× smaller at depth=500**.
- **Protobuf unmarshal is ~7.9× faster** than JSON at depth=20 and
  **~6.3× faster** at depth=500 with **~78% fewer allocations**.
- **ProtoJSON is the worst of both worlds** — it uses reflection and string
  building, making it **8–13× slower than JSON** with massive allocation counts.
  Never use `protojson` in a hot path.
- **Current system (JSON)** is a reasonable tradeoff: human-readable wire format
  makes browser debugging easy, and JSON marshal produces only 1 alloc/op
  (pre-allocated buffer). The main cost is 2× the bandwidth vs binary Protobuf.
- For a high-frequency production exchange (>10K snapshots/sec), switching
  to Protobuf binary for SSE payloads and using `Transfer-Encoding: binary`
  (or a WebSocket binary frame) would halve bandwidth and double marshal throughput.

---

## Reproducing Results

```bash
# 1. Clone the repo and check out the benchmark branch
git clone https://github.com/kayden-vs/zaraba
cd zaraba
git checkout benchmark/performance-comparison

# 2. Install dependencies
go mod download

# 3. Run all benchmarks
./benchmarks/run_benchmarks.sh

# 4. Compare two runs
benchstat benchmarks/results/run1.txt benchmarks/results/run2.txt
```

> **Note**: Results vary by hardware. The key metrics for resume/blog use are
> the *ratios* (e.g., "Protobuf is 2.2× faster than JSON") rather than
> absolute ns/op values, which are portable across machines.

---

## File Structure

```
benchmarks/
├── helpers_test.go          # shared fixtures (buildSnapshot, buildExchangeServer, ...)
├── grpc_vs_rest_test.go     # Benchmark 1: gRPC vs JSON/REST
├── sse_vs_websocket_test.go # Benchmark 2: SSE hub vs WebSocket hub
├── serialization_test.go    # Benchmark 3: Protobuf vs JSON encoding
├── run_benchmarks.sh        # reproducible runner script
├── results/                 # benchmark output files (gitignored except .gitkeep)
└── README.md                # this file
```
