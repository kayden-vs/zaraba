#!/usr/bin/env bash
# run_benchmarks.sh — reproducible benchmark runner for zaraba
#
# Usage:
#   ./benchmarks/run_benchmarks.sh               # all suites, 5 runs
#   ./benchmarks/run_benchmarks.sh grpc          # only gRPC vs REST
#   ./benchmarks/run_benchmarks.sh sse           # only SSE vs WebSocket
#   ./benchmarks/run_benchmarks.sh serial        # only serialisation
#
# Results land in benchmarks/results/<timestamp>_<suite>.txt
# Install benchstat for statistical comparison:
#   go install golang.org/x/perf/cmd/benchstat@latest

set -euo pipefail

RESULTS_DIR="$(dirname "$0")/results"
mkdir -p "$RESULTS_DIR"
TS=$(date +%Y%m%d_%H%M%S)
COUNT=5
BENCHTIME=3s
TIMEOUT=20m

run() {
    local suite="$1"
    local pattern="$2"
    local out="$RESULTS_DIR/${TS}_${suite}.txt"
    echo "→ running $suite benchmarks (count=$COUNT, benchtime=$BENCHTIME)..."
    go test ./benchmarks/... \
        -bench="$pattern" \
        -benchmem \
        -benchtime="$BENCHTIME" \
        -count="$COUNT" \
        -timeout="$TIMEOUT" \
        | tee "$out"
    echo "   saved → $out"
}

case "${1:-all}" in
    grpc)   run grpc_vs_rest  'BenchmarkGRPC|BenchmarkREST' ;;
    sse)    run sse_vs_ws     'BenchmarkSSE|BenchmarkWS'    ;;
    serial) run serialization 'BenchmarkJSON|BenchmarkProtobuf|BenchmarkProtoJSON' ;;
    all)
        run grpc_vs_rest  'BenchmarkGRPC|BenchmarkREST'
        run sse_vs_ws     'BenchmarkSSE|BenchmarkWS'
        run serialization 'BenchmarkJSON|BenchmarkProtobuf|BenchmarkProtoJSON'
        ;;
    *)
        echo "unknown suite: $1  (use grpc | sse | serial | all)"
        exit 1
        ;;
esac

echo ""
echo "Done. To compare two runs with benchstat:"
echo "  benchstat results/<run1>.txt results/<run2>.txt"
