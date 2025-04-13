#!/bin/bash
# Script to run all benchmarks and tests and save the results

# Generate timestamp for filenames
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

# Create directory for benchmark results if it doesn't exist
mkdir -p benchmark_results

echo "Running standard benchmarks..."
go test -bench=. -benchmem -test.benchtime=1x -count=1 -v > benchmark_results/benchmarks-${TIMESTAMP}.txt

echo "Running all test cases..."
TEST_OUTPUT=benchmark_results/tests-${TIMESTAMP}.txt

# Run all the test cases and capture their output
echo "Running 10K test..."
go test -run=TestTenThousandKeys -v >> $TEST_OUTPUT

echo "Running UUID keys test..."
go test -run=TestUUIDKeys -v >> $TEST_OUTPUT

# Only run these if explicitly requested (they take longer)
if [ "$1" == "--full" ]; then
  echo "Running million keys test..."
  go test -run=TestMillionKeys -v >> $TEST_OUTPUT

  echo "Running 10 million keys test (this will take some time)..."
  go test -run=TestTenMillionKeys -v >> $TEST_OUTPUT
fi

# Now convert both outputs to JSON
echo "Converting benchmark results to JSON..."
go run benchmark_types.go benchmark_to_json.go benchmark_results/benchmarks-${TIMESTAMP}.txt

echo "Converting test results to JSON..."
go run benchmark_types.go benchmark_to_json.go $TEST_OUTPUT

# Merge the two sets of results into a combined file
echo "Creating combined results file..."
COMBINED_OUTPUT=benchmark_results/combined-${TIMESTAMP}.json
BENCH_JSON=benchmark_results/benchmarks-${TIMESTAMP}.json
TEST_JSON=${TEST_OUTPUT%.txt}.json

# Simple merge using jq if available, otherwise use a placeholder
if command -v jq >/dev/null 2>&1; then
  jq -s '.[0].results += .[1].results | .[0]' $BENCH_JSON $TEST_JSON > $COMBINED_OUTPUT
  echo "Combined JSON benchmark results written to $COMBINED_OUTPUT"
else
  echo "jq not found, cannot merge results. Individual JSON files are still available."
  cp $BENCH_JSON $COMBINED_OUTPUT
fi

echo "Benchmarks completed. Results saved in benchmark_results/ directory."
echo "To run full benchmarks including million-key tests, use: ./run_benchmarks.sh --full" 

# Create a timestamped copy of the latest.json file
# First determine the repository root directory regardless of where script is run from
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
REPO_ROOT="$( cd "$SCRIPT_DIR/../.." && pwd )"
BENCHMARK_HISTORY_DIR="$REPO_ROOT/benchmark_history"

# Make sure benchmark_history directory exists
mkdir -p "$BENCHMARK_HISTORY_DIR"

LATEST_JSON="$BENCHMARK_HISTORY_DIR/latest.json"
echo "Looking for latest.json at: $LATEST_JSON"

if [ -f "$LATEST_JSON" ]; then
  TIMESTAMP=$(date +%Y%m%d-%H%M%S)
  TIMESTAMPED_FILE="$BENCHMARK_HISTORY_DIR/benchmark-${TIMESTAMP}.json"
  echo "Creating timestamped benchmark file: ${TIMESTAMPED_FILE}"
  cp "$LATEST_JSON" "$TIMESTAMPED_FILE"
  echo "Timestamped benchmark file created successfully."
else
  echo "Warning: $LATEST_JSON not found. No timestamped copy created."
fi 