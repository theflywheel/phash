#!/bin/bash
# Simple script to compare two benchmark JSON files

set -e  # Exit on any error

if [ $# -ne 2 ]; then
  echo "Usage: $0 <current_json_file> <baseline_json_file>"
  exit 1
fi

CURRENT_JSON=$1
BASELINE_JSON=$2

if [ ! -f "$CURRENT_JSON" ]; then
  echo "Error: Current benchmark file not found: $CURRENT_JSON"
  exit 1
fi

if [ ! -f "$BASELINE_JSON" ]; then
  echo "Error: Baseline benchmark file not found: $BASELINE_JSON"
  exit 1
fi

echo "Comparing benchmark files:"
echo "  Current: $CURRENT_JSON"
echo "  Baseline: $BASELINE_JSON"

# Run the comparison using the Go comparison tool
cd "$(dirname "$0")"  # Move to tools directory
go run benchmark_types.go compare_benchmarks.go "../../$BASELINE_JSON" "../../$CURRENT_JSON"
COMPARISON_RESULT=$?
cd ../..

# Copy comparison output to benchmark history
mkdir -p benchmark_history
cp benchmark-comparison.json benchmark_history/pr-comparison-$(date +%Y%m%d-%H%M%S).json

# Output results
if [ $COMPARISON_RESULT -eq 0 ]; then
  echo "✅ No significant performance regressions detected!"
else
  echo "❌ Performance regression detected! See above for details."
fi

exit $COMPARISON_RESULT 