// Package main provides tools to compare benchmark results.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// BenchResult represents a single benchmark result with multiple metrics
type BenchResult struct {
	Name     string             `json:"name"`
	Category string             `json:"category"`
	Metrics  map[string]float64 `json:"metrics"`
}

// BenchSummary represents the complete benchmark output
type BenchSummary struct {
	Timestamp string        `json:"timestamp"`
	CommitID  string        `json:"commit_id"`
	Branch    string        `json:"branch"`
	GoVersion string        `json:"go_version"`
	System    string        `json:"system,omitempty"`
	Results   []BenchResult `json:"results"`
}

// MetricComparison represents a comparison between two metric values
type MetricComparison struct {
	Name          string  `json:"name"`
	BaseValue     float64 `json:"base_value"`
	CurrentValue  float64 `json:"current_value"`
	PercentChange float64 `json:"percent_change"`
	IsRegression  bool    `json:"is_regression"`
	IsImprovement bool    `json:"is_improvement"`
	IsSignificant bool    `json:"is_significant"`
}

// BenchmarkComparison represents a comparison between benchmark results
type BenchmarkComparison struct {
	Name              string             `json:"name"`
	Category          string             `json:"category"`
	MetricComparisons []MetricComparison `json:"metric_comparisons"`
	OverallAssessment string             `json:"overall_assessment"`
	HasRegressions    bool               `json:"has_regressions"`
	Score             float64            `json:"score"`
}

// ComparisonSummary represents the overall benchmark comparison result
type ComparisonSummary struct {
	BaseCommit             string                `json:"base_commit"`
	CurrentCommit          string                `json:"current_commit"`
	TotalBenchmarks        int                   `json:"total_benchmarks"`
	ImprovedBenchmarks     int                   `json:"improved_benchmarks"`
	RegressionBenchmarks   int                   `json:"regression_benchmarks"`
	SignificantRegressions int                   `json:"significant_regressions"`
	BenchmarkComparisons   []BenchmarkComparison `json:"benchmark_comparisons"`
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run compare_benchmarks.go <base_json_file> <current_json_file>")
		os.Exit(1)
	}

	baseFile := os.Args[1]
	currentFile := os.Args[2]

	// Load base benchmark data
	baseData, err := os.ReadFile(baseFile)
	if err != nil {
		fmt.Printf("Error reading base file: %v\n", err)
		os.Exit(1)
	}

	var baseSummary BenchSummary
	if err := json.Unmarshal(baseData, &baseSummary); err != nil {
		fmt.Printf("Error parsing base JSON: %v\n", err)
		os.Exit(1)
	}

	// Load current benchmark data
	currentData, err := os.ReadFile(currentFile)
	if err != nil {
		fmt.Printf("Error reading current file: %v\n", err)
		os.Exit(1)
	}

	var currentSummary BenchSummary
	if err := json.Unmarshal(currentData, &currentSummary); err != nil {
		fmt.Printf("Error parsing current JSON: %v\n", err)
		os.Exit(1)
	}

	// Create map of base results for quick lookup
	baseResults := make(map[string]BenchResult)
	for _, result := range baseSummary.Results {
		baseResults[result.Name] = result
	}

	// Compare results
	const significanceThreshold = 5.0 // 5% change threshold for marking as significant

	comparisons := []BenchmarkComparison{}
	significantRegressions := 0
	improvedBenchmarks := 0
	regressionBenchmarks := 0

	for _, currentResult := range currentSummary.Results {
		baseResult, found := baseResults[currentResult.Name]
		if !found {
			// Skip benchmarks not in base
			continue
		}

		benchmarkComparison := BenchmarkComparison{
			Name:              currentResult.Name,
			Category:          currentResult.Category,
			MetricComparisons: []MetricComparison{},
		}

		hasRegressions := false
		overallScore := 0.0
		totalMetrics := 0

		// Compare each metric
		for metricName, currentValue := range currentResult.Metrics {
			baseValue, found := baseResult.Metrics[metricName]
			if !found {
				continue
			}

			percentChange := 0.0
			if baseValue != 0 {
				percentChange = ((currentValue - baseValue) / baseValue) * 100
			}

			// Determine if this is a regression or improvement
			isRegression := false
			isImprovement := false
			isSignificant := false

			// For some metrics, higher is better (rates, operations)
			// For others, lower is better (ns/op, bytes/op, etc.)
			metricHigherIsBetter := isHigherBetterMetric(metricName)

			if metricHigherIsBetter {
				isRegression = percentChange < 0
				isImprovement = percentChange > 0
			} else {
				isRegression = percentChange > 0
				isImprovement = percentChange < 0
			}

			// Determine significance
			isSignificant = abs(percentChange) >= significanceThreshold

			// Track if this benchmark has any significant regressions
			if isRegression && isSignificant {
				hasRegressions = true
			}

			// Add to the overall score (improvements are positive, regressions are negative)
			if isImprovement {
				overallScore += abs(percentChange)
			} else if isRegression {
				overallScore -= abs(percentChange)
			}
			totalMetrics++

			metricComparison := MetricComparison{
				Name:          metricName,
				BaseValue:     baseValue,
				CurrentValue:  currentValue,
				PercentChange: percentChange,
				IsRegression:  isRegression,
				IsImprovement: isImprovement,
				IsSignificant: isSignificant,
			}

			benchmarkComparison.MetricComparisons = append(
				benchmarkComparison.MetricComparisons,
				metricComparison)
		}

		// Calculate the average score
		if totalMetrics > 0 {
			benchmarkComparison.Score = overallScore / float64(totalMetrics)
		}

		// Set overall assessment
		benchmarkComparison.HasRegressions = hasRegressions
		if hasRegressions {
			benchmarkComparison.OverallAssessment = "REGRESSION"
			regressionBenchmarks++
			if hasRegressions {
				significantRegressions++
			}
		} else if benchmarkComparison.Score > 0 {
			benchmarkComparison.OverallAssessment = "IMPROVEMENT"
			improvedBenchmarks++
		} else {
			benchmarkComparison.OverallAssessment = "NEUTRAL"
		}

		comparisons = append(comparisons, benchmarkComparison)
	}

	// Sort comparisons (worst regressions first)
	sort.Slice(comparisons, func(i, j int) bool {
		// First priority: regressions before non-regressions
		if comparisons[i].HasRegressions != comparisons[j].HasRegressions {
			return comparisons[i].HasRegressions
		}
		// Second priority: sort by score (lower/worse score first)
		return comparisons[i].Score < comparisons[j].Score
	})

	// Create summary
	summary := ComparisonSummary{
		BaseCommit:             baseSummary.CommitID,
		CurrentCommit:          currentSummary.CommitID,
		TotalBenchmarks:        len(comparisons),
		ImprovedBenchmarks:     improvedBenchmarks,
		RegressionBenchmarks:   regressionBenchmarks,
		SignificantRegressions: significantRegressions,
		BenchmarkComparisons:   comparisons,
	}

	// Output summary
	printComparisonSummary(summary)

	// Write JSON comparison to file
	outputPath := "benchmark-comparison.json"
	jsonData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		fmt.Printf("Error creating comparison JSON: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		fmt.Printf("Error writing comparison file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Comparison JSON written to %s\n", outputPath)

	// Exit with non-zero code if there are significant regressions
	if significantRegressions > 0 {
		fmt.Printf("\n⚠️ WARNING: %d significant performance regressions detected!\n", significantRegressions)
		os.Exit(1)
	}
}

// printComparisonSummary outputs a human-readable comparison report
func printComparisonSummary(summary ComparisonSummary) {
	fmt.Printf("Benchmark Comparison: %s vs %s\n\n",
		truncateString(summary.BaseCommit, 8),
		truncateString(summary.CurrentCommit, 8))

	fmt.Printf("Summary:\n")
	fmt.Printf("- Total benchmarks compared: %d\n", summary.TotalBenchmarks)
	fmt.Printf("- Improvements: %d\n", summary.ImprovedBenchmarks)
	fmt.Printf("- Regressions: %d (significant: %d)\n\n",
		summary.RegressionBenchmarks, summary.SignificantRegressions)

	if summary.TotalBenchmarks == 0 {
		fmt.Println("No matching benchmarks found for comparison")
		return
	}

	fmt.Println("Benchmark Details (sorted by impact):")
	fmt.Println("======================================")

	for _, comp := range summary.BenchmarkComparisons {
		// Add emoji indicator for quick visual feedback
		indicator := "✅" // Improvement
		if comp.HasRegressions {
			indicator = "❌" // Regression
		} else if comp.Score < 0 {
			indicator = "⚠️" // Minor regression but not significant
		} else if comp.Score == 0 {
			indicator = "⏺" // Neutral
		}

		fmt.Printf("\n%s %s (%s):\n", indicator, comp.Name, comp.Category)

		// Show the most impactful metrics first
		sort.Slice(comp.MetricComparisons, func(i, j int) bool {
			return abs(comp.MetricComparisons[i].PercentChange) >
				abs(comp.MetricComparisons[j].PercentChange)
		})

		for _, metric := range comp.MetricComparisons {
			// Skip metrics with no change
			if metric.PercentChange == 0 {
				continue
			}

			metricIndicator := " "
			if metric.IsRegression && metric.IsSignificant {
				metricIndicator = "▼" // Significant regression
			} else if metric.IsImprovement && metric.IsSignificant {
				metricIndicator = "▲" // Significant improvement
			}

			fmt.Printf("  %s %-20s: %+8.2f%% (%g → %g)\n",
				metricIndicator,
				metric.Name,
				metric.PercentChange,
				metric.BaseValue,
				metric.CurrentValue)
		}
	}
}

// Helper functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// isHigherBetterMetric determines if a higher value is better for a given metric
func isHigherBetterMetric(metricName string) bool {
	// For these metrics, higher values are better
	higherBetterMetrics := []string{
		"ops_per_sec", "operations", "insertion_rate", "lookup_rate",
		"sequential_lookup_rate", "random_lookup_rate", "batch_",
		"rate_", "max_", "throughput",
	}

	// Check if metric name contains any of the higher-is-better patterns
	for _, pattern := range higherBetterMetrics {
		if strings.Contains(metricName, pattern) {
			return true
		}
	}

	// Default: lower is better (ns/op, bytes/op, allocs/op, etc.)
	return false
}
