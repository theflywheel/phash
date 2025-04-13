#!/bin/bash
# Script to run benchmarks and store results in JSON format

# Create benchmark history directory if it doesn't exist
BENCHMARK_DIR="benchmark_history"
mkdir -p $BENCHMARK_DIR

# Get git information
if git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
  COMMIT_ID=$(git rev-parse HEAD)
  BRANCH=$(git rev-parse --abbrev-ref HEAD)
else
  echo "Not in a git repository, using default values"
  COMMIT_ID="local"
  BRANCH="dev"
fi
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

echo "Running benchmarks for commit $COMMIT_ID on branch $BRANCH..."

# Run benchmarks
cd bench

# Run standard benchmarks - they'll automatically output to JSON
echo "Running standard benchmarks..."
go test -bench=. -benchmem -benchtime=1x -v

# Clean up intermediate files
echo "Cleaning up temporary files..."
cd ..
rm -f $BENCHMARK_DIR/tmp_*

echo "Benchmark process completed successfully. Results saved to $BENCHMARK_DIR/latest.json"
echo "A timestamped copy is also stored in the same directory." 