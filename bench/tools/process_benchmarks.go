package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Process both benchmark files
	standardFile := "../../benchmark_history/standard_benchmarks.txt"
	scaleFile := "../../benchmark_history/scale_benchmarks.txt"

	// Read benchmark outputs
	standardData, err := os.ReadFile(standardFile)
	if err != nil {
		fmt.Printf("Error reading standard benchmark file: %v\n", err)
		os.Exit(1)
	}

	scaleData, err := os.ReadFile(scaleFile)
	if err != nil {
		fmt.Printf("Error reading scale benchmark file: %v\n", err)
		os.Exit(1)
	}

	// Create the combined results structure
	results := []map[string]interface{}{}

	// Extract standard Go benchmarks from both files
	stdBenchRegex := regexp.MustCompile(`Benchmark(\w+)(?:-\d+)?\s+(\d+)\s+(\d+\.?\d*)\s+ns/op(?:\s+(\d+)\s+B/op)?(?:\s+(\d+)\s+allocs/op)?`)

	// Process both files for benchmark results
	standardMatches := stdBenchRegex.FindAllStringSubmatch(string(standardData), -1)
	scaleMatches := stdBenchRegex.FindAllStringSubmatch(string(scaleData), -1)

	// Combine all matches
	allMatches := append(standardMatches, scaleMatches...)

	for _, matches := range allMatches {
		name := matches[1]
		ops, _ := strconv.Atoi(matches[2])
		nsPerOp, _ := strconv.ParseFloat(matches[3], 64)

		result := map[string]interface{}{
			"name":       name,
			"operations": ops,
			"ns_per_op":  nsPerOp,
		}

		// Add bytes_per_op if present
		if len(matches) > 4 && matches[4] != "" {
			bytesPerOp, _ := strconv.Atoi(matches[4])
			result["bytes_per_op"] = bytesPerOp
		}

		// Add allocs_per_op if present
		if len(matches) > 5 && matches[5] != "" {
			allocsPerOp, _ := strconv.Atoi(matches[5])
			result["allocs_per_op"] = allocsPerOp
		}

		// Set category based on benchmark name
		if name == "Put" || name == "Get" || name == "SimplePut" || name == "SimpleGet" {
			result["category"] = "standard"
		} else {
			result["category"] = "scale"

			// Extract metrics from log output
			scaleContent := string(scaleData)
			metrics := map[string]float64{}

			// Extract metrics based on benchmark type
			if name == "TenThousandKeys" {
				metrics = extractMetricsForPattern(scaleContent, "Time to insert 10000 keys:", "keys/sec", "insertion_rate")
				metrics2 := extractMetricsForPattern(scaleContent, "Time to perform 1000 random lookups:", "lookups/sec", "random_lookup_rate")
				for k, v := range metrics2 {
					metrics[k] = v
				}
				metrics3 := extractMetricsForPattern(scaleContent, "Time to verify all 10000 keys:", "keys/sec", "sequential_lookup_rate")
				for k, v := range metrics3 {
					metrics[k] = v
				}
				metrics4 := extractMetricsForPattern(scaleContent, "Average bytes per key-value pair:", "bytes", "bytes_per_key")
				if len(metrics4) == 0 {
					metrics4 = extractMetricsForPattern(scaleContent, "Average bytes per key-value pair", "bytes", "bytes_per_key")
				}
				for k, v := range metrics4 {
					metrics[k] = v
				}
				metrics5 := extractMetricsForPattern(scaleContent, "File size for 10000 keys:", "MB", "file_size_mb")
				for k, v := range metrics5 {
					metrics[k] = v
				}
			} else if name == "MillionKeys" {
				metrics = extractMetricsForPattern(scaleContent, "Time to insert 1000000 keys:", "keys/sec", "insertion_rate")
				metrics2 := extractMetricsForPattern(scaleContent, "Time to verify 10000 sampled keys:", "keys/sec", "verification_rate")
				for k, v := range metrics2 {
					metrics[k] = v
				}
				metrics3 := extractMetricsForPattern(scaleContent, "Average bytes per key-value pair:", "bytes", "bytes_per_key")
				if len(metrics3) == 0 {
					metrics3 = extractMetricsForPattern(scaleContent, "Average bytes per key-value pair", "bytes", "bytes_per_key")
				}
				for k, v := range metrics3 {
					metrics[k] = v
				}
				metrics4 := extractMetricsForPattern(scaleContent, "File size for 1000000 keys:", "MB", "file_size_mb")
				for k, v := range metrics4 {
					metrics[k] = v
				}
			} else if name == "TenMillionKeys" {
				metrics = extractMetricsForPattern(scaleContent, "Time to insert 10000000 keys:", "keys/sec", "insertion_rate")
				metrics2 := extractMetricsForPattern(scaleContent, "Time to perform 100000 random lookups:", "lookups/sec", "random_lookup_rate")
				for k, v := range metrics2 {
					metrics[k] = v
				}
				metrics3 := extractMetricsForPattern(scaleContent, "Average bytes per key-value pair:", "bytes", "bytes_per_key")
				if len(metrics3) == 0 {
					metrics3 = extractMetricsForPattern(scaleContent, "Average bytes per key-value pair", "bytes", "bytes_per_key")
				}
				for k, v := range metrics3 {
					metrics[k] = v
				}
				metrics4 := extractMetricsForPattern(scaleContent, "File size for 10000000 keys:", "MB", "file_size_mb")
				for k, v := range metrics4 {
					metrics[k] = v
				}
			} else if name == "UUIDKeys" {
				metrics = extractMetricsForPattern(scaleContent, "Time to insert 100000 UUID keys:", "keys/sec", "insertion_rate")
				metrics2 := extractMetricsForPattern(scaleContent, "Time to retrieve 100000 UUID keys", "keys/sec", "retrieval_rate")
				for k, v := range metrics2 {
					metrics[k] = v
				}
				metrics3 := extractMetricsForPattern(scaleContent, "Time to validate 100000 UUID keys:", "keys/sec", "validation_rate")
				for k, v := range metrics3 {
					metrics[k] = v
				}
				metrics4 := extractMetricsForPattern(scaleContent, "Average bytes per key-value pair:", "bytes", "bytes_per_key")
				if len(metrics4) == 0 {
					metrics4 = extractMetricsForPattern(scaleContent, "Average bytes per key-value pair", "bytes", "bytes_per_key")
				}
				for k, v := range metrics4 {
					metrics[k] = v
				}
				metrics5 := extractMetricsForPattern(scaleContent, "File size for 100000 UUID keys:", "MB", "file_size_mb")
				for k, v := range metrics5 {
					metrics[k] = v
				}
			}

			if len(metrics) > 0 {
				result["metrics"] = metrics
			}
		}

		results = append(results, result)
	}

	// Create summary object
	summary := map[string]interface{}{
		"timestamp":  time.Now().Format(time.RFC3339),
		"commit_id":  "current",
		"branch":     "main",
		"go_version": extractGoVersion(string(standardData) + string(scaleData)),
		"results":    results,
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		fmt.Printf("Error creating JSON: %v\n", err)
		os.Exit(1)
	}

	// Write baseline and latest files
	baselinePath := "../../benchmark_history/baseline.json"
	latestPath := "../../benchmark_history/latest.json"

	err = os.WriteFile(baselinePath, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing baseline file: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(latestPath, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing latest file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created benchmark files:\n")
	fmt.Printf("  - %s\n", baselinePath)
	fmt.Printf("  - %s\n", latestPath)
}

// Extract the Go version from benchmark output
func extractGoVersion(content string) string {
	goVersionRegex := regexp.MustCompile(`go\d+\.\d+(?:\.\d+)?`)
	if match := goVersionRegex.FindString(content); match != "" {
		return match
	}
	return "go1.x"
}

// Extract metric values from log output
func extractMetricsForPattern(content, pattern, suffix, metricName string) map[string]float64 {
	metrics := map[string]float64{}

	// Handle both value formats: with parentheses and without
	regexWithParens := regexp.MustCompile(pattern + `.*\(([\d,\.]+) ` + suffix + `\)`)
	regexWithoutParens := regexp.MustCompile(pattern + `\s+([\d,\.]+)\s+` + suffix)

	// Try with parentheses first
	if matches := regexWithParens.FindStringSubmatch(content); len(matches) > 1 {
		// Convert value string to float
		valueStr := strings.ReplaceAll(matches[1], ",", "")
		if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
			metrics[metricName] = value
		}
	} else if matches := regexWithoutParens.FindStringSubmatch(content); len(matches) > 1 {
		// Try without parentheses
		valueStr := strings.ReplaceAll(matches[1], ",", "")
		if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
			metrics[metricName] = value
		}
	}

	return metrics
}
