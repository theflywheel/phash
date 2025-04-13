#!/bin/bash

# Clear the latest.json file
rm -f benchmark_history/latest.json

# Run all benchmarks in a single command to ensure they're all captured in the same results
echo "Running all benchmarks..."
go test -bench='Benchmark' -count=1 -test.benchtime=1x -run=^$ ./bench/

echo "All benchmarks completed!"

# Create a timestamped copy of the latest.json file
LATEST_JSON="benchmark_history/latest.json"
if [ -f "$LATEST_JSON" ]; then
  TIMESTAMP=$(date +%Y%m%d-%H%M%S)
  TIMESTAMPED_FILE="benchmark_history/benchmark-${TIMESTAMP}.json"
  cp "$LATEST_JSON" "$TIMESTAMPED_FILE"
  echo "Created timestamped benchmark file: ${TIMESTAMPED_FILE}"
else
  echo "Warning: $LATEST_JSON not found. No timestamped copy created."
fi 