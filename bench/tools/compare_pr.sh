#!/bin/bash
# Script to compare PR benchmarks against the latest main branch benchmark

set -e  # Exit on any error

# Create benchmark directory if it doesn't exist
BENCHMARK_DIR="benchmark_history"
mkdir -p $BENCHMARK_DIR

# Get git information for PR branch
PR_COMMIT_ID=$(git rev-parse HEAD)
PR_BRANCH=$(git rev-parse --abbrev-ref HEAD)
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

# Create benchmark output files with git info in name
PR_BENCHMARK_FILE="$BENCHMARK_DIR/pr-benchmark-${PR_BRANCH}-${PR_COMMIT_ID:0:8}-${TIMESTAMP}.txt"
PR_TEST_FILE="$BENCHMARK_DIR/pr-tests-${PR_BRANCH}-${PR_COMMIT_ID:0:8}-${TIMESTAMP}.txt"
PR_COMBINED_FILE="$BENCHMARK_DIR/pr-combined-${PR_BRANCH}-${PR_COMMIT_ID:0:8}-${TIMESTAMP}.json"

echo "Running benchmarks for PR commit $PR_COMMIT_ID on branch $PR_BRANCH..."

# Run benchmarks for PR branch
cd bench

# Run standard benchmarks
echo "Running standard benchmarks..."
go test -bench=. -benchmem -v > "../$PR_BENCHMARK_FILE"

# Run all test cases
echo "Running test cases..."
echo "Running 10K test..."
go test -run=TestTenThousandKeys -v > "../$PR_TEST_FILE"

echo "Running UUID keys test..."
go test -run=TestUUIDKeys -v >> "../$PR_TEST_FILE"

# Run million keys test if --full flag is provided
if [ "$1" == "--full" ]; then
  echo "Running million keys test..."
  go test -run=TestMillionKeys -v >> "../$PR_TEST_FILE"
  
  echo "Running 10 million keys test (this will take some time)..."
  go test -run=TestTenMillionKeys -v >> "../$PR_TEST_FILE"
fi

echo "PR benchmarks completed, converting to JSON..."

# Convert benchmark to JSON with git information
cd tools
go run benchmark_types.go benchmark_to_json.go "../../$PR_BENCHMARK_FILE" "$PR_COMMIT_ID" "$PR_BRANCH"
go run benchmark_types.go benchmark_to_json.go "../../$PR_TEST_FILE" "$PR_COMMIT_ID" "$PR_BRANCH"

# Get the JSON filenames
PR_BENCHMARK_JSON="${PR_BENCHMARK_FILE%.txt}.json"
PR_TEST_JSON="${PR_TEST_FILE%.txt}.json"

# Merge the results if jq is available
if command -v jq >/dev/null 2>&1; then
  echo "Merging benchmark and test results..."
  jq -s '.[0].results += .[1].results | .[0]' "../../$PR_BENCHMARK_JSON" "../../$PR_TEST_JSON" > "../../$PR_COMBINED_FILE"
  echo "Combined results stored at: $PR_COMBINED_FILE"
else
  echo "jq not found, copying benchmark results as combined results..."
  cp "../../$PR_BENCHMARK_JSON" "../../$PR_COMBINED_FILE"
fi

cd ../..

# Find the latest main branch benchmark
MAIN_JSON_FILE="$BENCHMARK_DIR/latest.json"

if [ ! -f "$MAIN_JSON_FILE" ]; then
  echo "Error: No baseline benchmark found for comparison."
  echo "Please run benchmarks on the main branch first."
  exit 1
fi

echo "Comparing PR benchmarks against main branch baseline..."

# Run the comparison
cd bench/tools
go run benchmark_types.go compare_benchmarks.go "../../$MAIN_JSON_FILE" "../../$PR_COMBINED_FILE"
COMPARISON_RESULT=$?
cd ../..

# Copy comparison output to benchmark history
cp benchmark-comparison.json "$BENCHMARK_DIR/pr-comparison-${PR_BRANCH}-${PR_COMMIT_ID:0:8}-${TIMESTAMP}.json"

# Output results
if [ $COMPARISON_RESULT -eq 0 ]; then
  echo "✅ No significant performance regressions detected!"
else
  echo "❌ Performance regression detected! See above for details."
fi

exit $COMPARISON_RESULT 