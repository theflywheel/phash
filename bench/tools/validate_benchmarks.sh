#!/bin/bash
# Script to validate benchmark results for CI
# Returns non-zero exit code if benchmarks have degraded significantly

set -e  # Exit on any error

COMPARISON_FILE="benchmark-comparison.json"

if [ ! -f "$COMPARISON_FILE" ]; then
  echo "Error: Benchmark comparison file not found at $COMPARISON_FILE"
  echo "Run benchmark comparison first."
  exit 1
fi

echo "Analyzing benchmark comparison results..."

# Extract key metrics from the comparison JSON
SIGNIFICANT_REGRESSIONS=$(jq '.significant_regressions' "$COMPARISON_FILE")
TOTAL_BENCHMARKS=$(jq '.total_benchmarks' "$COMPARISON_FILE")
IMPROVED_BENCHMARKS=$(jq '.improved_benchmarks' "$COMPARISON_FILE")
REGRESSION_BENCHMARKS=$(jq '.regression_benchmarks' "$COMPARISON_FILE")

echo "Summary:"
echo "- Total benchmarks: $TOTAL_BENCHMARKS"
echo "- Improved benchmarks: $IMPROVED_BENCHMARKS"
echo "- Regression benchmarks: $REGRESSION_BENCHMARKS"
echo "- Significant regressions: $SIGNIFICANT_REGRESSIONS"

# Create improved flag for PR comment
if [ "$SIGNIFICANT_REGRESSIONS" -eq 0 ]; then
  echo '{"improved": true, "benchmarks": '$(jq '.benchmark_comparisons | map({name: .name, baseline_ns_per_op: (.metric_comparisons[] | select(.name == "ns/op") | .base_value), current_ns_per_op: (.metric_comparisons[] | select(.name == "ns/op") | .current_value), percent_change: (.metric_comparisons[] | select(.name == "ns/op") | .percent_change)})' "$COMPARISON_FILE")'}' > benchmark-summary.json
else
  echo '{"improved": false, "benchmarks": '$(jq '.benchmark_comparisons | map({name: .name, baseline_ns_per_op: (.metric_comparisons[] | select(.name == "ns/op") | .base_value), current_ns_per_op: (.metric_comparisons[] | select(.name == "ns/op") | .current_value), percent_change: (.metric_comparisons[] | select(.name == "ns/op") | .percent_change)})' "$COMPARISON_FILE")'}' > benchmark-summary.json
fi

# Create a symlink for the GitHub Action to use
ln -sf benchmark-summary.json benchmark-comparison.json

# Decide if we should fail CI based on significant regressions
if [ "$SIGNIFICANT_REGRESSIONS" -gt 0 ]; then
  echo "❌ Found $SIGNIFICANT_REGRESSIONS significant benchmark regressions!"
  echo "CI validation failed - performance has degraded."
  exit 1
else
  if [ "$IMPROVED_BENCHMARKS" -gt 0 ]; then
    echo "✅ Benchmarks have improved!"
  else
    echo "✅ No significant benchmark regressions detected."
  fi
  echo "CI validation passed."
  exit 0
fi 