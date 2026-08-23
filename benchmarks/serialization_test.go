package benchmarks

// Benchmark 3: Protobuf binary vs JSON serialisation for orderbook snapshots.
//
// Every SSE broadcast marshals a pb.OrderbookSnapshot to JSON before writing
// "data: <json>\n\n" to the wire. Protobuf binary encoding is the alternative
// used in gRPC streaming — smaller payloads, fewer allocations, faster parsing.
//
// Metrics captured per snapshot depth (number of price levels):
//   - Marshal / Unmarshal time (ns/op)
//   - Serialised payload size in bytes — via b.ReportMetric
//   - Allocations (B/op, allocs/op)

import (
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ── JSON (encoding/json) ─────────────────────────────────────────────────────

func BenchmarkJSON_Marshal_Depth20(b *testing.B)  { benchmarkJSONMarshal(b, 20) }
func BenchmarkJSON_Marshal_Depth100(b *testing.B) { benchmarkJSONMarshal(b, 100) }
func BenchmarkJSON_Marshal_Depth500(b *testing.B) { benchmarkJSONMarshal(b, 500) }

func BenchmarkJSON_Unmarshal_Depth20(b *testing.B)  { benchmarkJSONUnmarshal(b, 20) }
func BenchmarkJSON_Unmarshal_Depth100(b *testing.B) { benchmarkJSONUnmarshal(b, 100) }
func BenchmarkJSON_Unmarshal_Depth500(b *testing.B) { benchmarkJSONUnmarshal(b, 500) }

func benchmarkJSONMarshal(b *testing.B, depth int) {
	s := buildSnapshot(depth)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf, _ := json.Marshal(s)
		if i == 0 {
			b.ReportMetric(float64(len(buf)), "bytes/payload")
		}
		_ = buf
	}
}

func benchmarkJSONUnmarshal(b *testing.B, depth int) {
	s := buildSnapshot(depth)
	buf, _ := json.Marshal(s)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var dst map[string]any
		_ = json.Unmarshal(buf, &dst)
	}
}

// ── Protobuf binary (google.golang.org/protobuf/proto) ───────────────────────

func BenchmarkProtobuf_Marshal_Depth20(b *testing.B)  { benchmarkProtobufMarshal(b, 20) }
func BenchmarkProtobuf_Marshal_Depth100(b *testing.B) { benchmarkProtobufMarshal(b, 100) }
func BenchmarkProtobuf_Marshal_Depth500(b *testing.B) { benchmarkProtobufMarshal(b, 500) }

func BenchmarkProtobuf_Unmarshal_Depth20(b *testing.B)  { benchmarkProtobufUnmarshal(b, 20) }
func BenchmarkProtobuf_Unmarshal_Depth100(b *testing.B) { benchmarkProtobufUnmarshal(b, 100) }
func BenchmarkProtobuf_Unmarshal_Depth500(b *testing.B) { benchmarkProtobufUnmarshal(b, 500) }

func benchmarkProtobufMarshal(b *testing.B, depth int) {
	s := buildSnapshot(depth)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf, _ := proto.Marshal(s)
		if i == 0 {
			b.ReportMetric(float64(len(buf)), "bytes/payload")
		}
		_ = buf
	}
}

func benchmarkProtobufUnmarshal(b *testing.B, depth int) {
	s := buildSnapshot(depth)
	buf, _ := proto.Marshal(s)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		clone := s.ProtoReflect().New().Interface()
		_ = proto.Unmarshal(buf, clone)
	}
}

// ── protojson (human-readable JSON via Protobuf reflection) ──────────────────

// BenchmarkProtoJSON_* measures protojson — the format used when proto structs
// are passed to encoding/json (they implement json.Marshaler via protojson).

func BenchmarkProtoJSON_Marshal_Depth20(b *testing.B)  { benchmarkProtoJSONMarshal(b, 20) }
func BenchmarkProtoJSON_Marshal_Depth100(b *testing.B) { benchmarkProtoJSONMarshal(b, 100) }
func BenchmarkProtoJSON_Marshal_Depth500(b *testing.B) { benchmarkProtoJSONMarshal(b, 500) }

func BenchmarkProtoJSON_Unmarshal_Depth20(b *testing.B)  { benchmarkProtoJSONUnmarshal(b, 20) }
func BenchmarkProtoJSON_Unmarshal_Depth100(b *testing.B) { benchmarkProtoJSONUnmarshal(b, 100) }
func BenchmarkProtoJSON_Unmarshal_Depth500(b *testing.B) { benchmarkProtoJSONUnmarshal(b, 500) }

func benchmarkProtoJSONMarshal(b *testing.B, depth int) {
	s := buildSnapshot(depth)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf, _ := protojson.Marshal(s)
		if i == 0 {
			b.ReportMetric(float64(len(buf)), "bytes/payload")
		}
		_ = buf
	}
}

func benchmarkProtoJSONUnmarshal(b *testing.B, depth int) {
	s := buildSnapshot(depth)
	buf, _ := protojson.Marshal(s)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		clone := s.ProtoReflect().New().Interface()
		_ = protojson.Unmarshal(buf, clone)
	}
}
