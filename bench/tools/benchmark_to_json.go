// Package main provides tools to parse and format benchmark results.
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

// BenchResult represents a test or benchmark result with multiple metrics.
type BenchResult struct {
	Name        string             `json:"name"`
	Category    string             `json:"category,omitempty"` // "standard", "scale", "uuid", etc.
	Description string             `json:"description,omitempty"`
	Metrics     map[string]float64 `json:"metrics"`
	RawOutput   string             `json:"raw_output,omitempty"`
}

// BenchSummary represents all benchmark results.
type BenchSummary struct {
	Timestamp  string        `json:"timestamp"`
	CommitID   string        `json:"commit_id"`
	Branch     string        `json:"branch"`
	GoVersion  string        `json:"go_version"`
	SystemInfo string        `json:"system_info,omitempty"`
	Results    []BenchResult `json:"results"`
}

// Main function to parse benchmark output and convert to JSON.
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run benchmark_to_json.go <benchmark_output_file> [commit_id] [branch_name]")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	commitID := "unknown"
	branch := "unknown"

	if len(os.Args) >= 3 {
		commitID = os.Args[2]
	}

	if len(os.Args) >= 4 {
		branch = os.Args[3]
	}

	// Read benchmark output
	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	content := string(data)
	results := []BenchResult{}
	goVersion := ""
	systemInfo := ""

	// Extract system info
	if sysMatch := regexp.MustCompile(`goos:.+goarch:.+`).FindString(content); sysMatch != "" {
		systemInfo = strings.TrimSpace(sysMatch)
	}

	// Find Go version
	if verMatch := regexp.MustCompile(`go\d+\.\d+(?:\.\d+)?`).FindString(content); verMatch != "" {
		goVersion = verMatch
	}

	// Classify benchmarks by type
	standardBenchmarks := map[string]bool{
		"Put":       true,
		"Get":       true,
		"SimplePut": true,
		"SimpleGet": true,
	}

	scaleBenchmarks := map[string]bool{
		"TenThousandKeys": true,
		"MillionKeys":     true,
		"TenMillionKeys":  true,
		"UUIDKeys":        true,
	}

	// Extract standard Go benchmarks
	stdBenchRegex := regexp.MustCompile(`Benchmark(\w+)(?:-\d+)?\s+(\d+)\s+(\d+\.?\d*)\s+ns/op(?:\s+(\d+)\s+B/op)?(?:\s+(\d+)\s+allocs/op)?`)
	for _, matches := range stdBenchRegex.FindAllStringSubmatch(content, -1) {
		name := matches[1]
		ops, _ := strconv.Atoi(matches[2])
		nsPerOp, _ := strconv.ParseFloat(matches[3], 64)

		metrics := map[string]float64{
			"operations": float64(ops),
			"ns_per_op":  nsPerOp,
		}

		// Only add ops_per_sec for standard benchmarks, not for scale benchmarks
		// which have their own rate metrics
		if !scaleBenchmarks[name] {
			metrics["ops_per_sec"] = 1_000_000_000 / nsPerOp
		}

		if len(matches) > 4 && matches[4] != "" {
			bytesPerOp, _ := strconv.Atoi(matches[4])
			metrics["bytes_per_op"] = float64(bytesPerOp)
		}

		if len(matches) > 5 && matches[5] != "" {
			allocsPerOp, _ := strconv.Atoi(matches[5])
			metrics["allocs_per_op"] = float64(allocsPerOp)
		}

		// Determine category based on benchmark name
		category := "other"
		if standardBenchmarks[name] {
			category = "standard"
		} else if scaleBenchmarks[name] {
			category = "scale"
		}

		result := BenchResult{
			Name:     name,
			Category: category,
			Metrics:  metrics,
		}

		results = append(results, result)
	}

	// Extract scale test results
	scaleTestRegex := regexp.MustCompile(`(?m)^=+\s+([\w\s]+)\s+Benchmark\s+Summary\s+=+$`)
	summaryBlocks := scaleTestRegex.FindAllStringSubmatchIndex(content, -1)

	// Process each summary block
	for i, block := range summaryBlocks {
		// Extract the benchmark name from the summary header
		benchName := content[block[2]:block[3]]
		benchName = strings.TrimSpace(strings.ReplaceAll(benchName, "Benchmark Summary", ""))

		// Determine the summary content (to next summary or end of file)
		var summaryContent string
		if i < len(summaryBlocks)-1 {
			summaryContent = content[block[0]:summaryBlocks[i+1][0]]
		} else {
			summaryContent = content[block[0]:]
		}

		// Extract metrics from the summary
		metrics := extractMetricsFromSummary(summaryContent)

		// Also look for the test pass line to get the test duration
		testPassRegex := regexp.MustCompile(`--- PASS: Test(\w+)\s+\((\d+\.\d+)s\)`)
		if passMatch := testPassRegex.FindStringSubmatch(content); len(passMatch) > 2 {
			testName := passMatch[1]
			duration, _ := strconv.ParseFloat(passMatch[2], 64)

			// Only add if this matches our benchmark name
			if strings.Contains(benchName, testName) {
				metrics["total_time_sec"] = duration
			}
		}

		result := BenchResult{
			Name:     benchName,
			Category: "scale",
			Metrics:  metrics,
		}

		results = append(results, result)
	}

	// Extract all detailed metrics from the entire output
	lineMetrics := extractMetricsFromRawOutput(content)

	// First try direct benchmark name matching for benchmark-specific metrics
	for i, result := range results {
		if result.Category == "scale" {
			// Look for metrics specifically generated for this benchmark
			benchPrefix := result.Name + "_"
			for metricName, value := range lineMetrics {
				if strings.HasPrefix(metricName, benchPrefix) {
					// Extract the actual metric name without the benchmark prefix
					actualMetric := strings.TrimPrefix(metricName, benchPrefix)
					results[i].Metrics[actualMetric] = value
					// Delete the prefixed metric to avoid duplicate processing
					delete(lineMetrics, metricName)
				}
			}
		}
	}

	// Process remaining metrics
	for metricName, value := range lineMetrics {
		// Determine which benchmark this belongs to
		assigned := false

		// First try direct benchmark name matching for specific rate metrics
		if strings.HasPrefix(metricName, "rate_") ||
			strings.HasPrefix(metricName, "batch_") ||
			strings.HasPrefix(metricName, "bytes_per") ||
			strings.HasPrefix(metricName, "filesize_mb_") {

			// TenThousandKeys metrics
			if strings.Contains(metricName, "random") ||
				strings.Contains(metricName, "sequential") ||
				strings.Contains(metricName, "verify") {
				// Sequential and random lookup rates are from TenThousandKeys
				for i, result := range results {
					if result.Name == "TenThousandKeys" {
						results[i].Metrics[metricName] = value
						assigned = true
						break
					}
				}
			} else if strings.Contains(metricName, "key_value") ||
				strings.Contains(metricName, "pair") {
				// Key-value pair metrics for all scale benchmarks
				for i, result := range results {
					if result.Category == "scale" {
						results[i].Metrics["bytes_per_key"] = value
					}
				}
				assigned = true
			} else if strings.Contains(metricName, "validate") ||
				strings.Contains(metricName, "retrieve") {
				// UUID benchmark metrics
				for i, result := range results {
					if result.Name == "UUIDKeys" {
						results[i].Metrics[metricName] = value
						assigned = true
						break
					}
				}
			} else if strings.Contains(metricName, "insert") {
				// TenThousandKeys gets the batch_insert metrics
				for i, result := range results {
					if result.Name == "TenThousandKeys" {
						results[i].Metrics[metricName] = value
						assigned = true
						break
					}
				}
			}
		}

		// If still not assigned, try the benchmark category matching
		if !assigned {
			// For each scale benchmark, assign metrics based on matching patterns
			for i, result := range results {
				if result.Category == "scale" {
					switch result.Name {
					case "TenThousandKeys":
						if strings.Contains(metricName, "thousand") ||
							strings.Contains(metricName, "10k") ||
							strings.Contains(metricName, "ten_thousand") {
							results[i].Metrics[metricName] = value
							assigned = true
						}
					case "MillionKeys":
						if strings.Contains(metricName, "million") &&
							!strings.Contains(metricName, "ten") {
							results[i].Metrics[metricName] = value
							assigned = true
						}
					case "TenMillionKeys":
						if strings.Contains(metricName, "ten_million") ||
							strings.Contains(metricName, "10million") {
							results[i].Metrics[metricName] = value
							assigned = true
						}
					case "UUIDKeys":
						if strings.Contains(metricName, "uuid") {
							results[i].Metrics[metricName] = value
							assigned = true
						}
					}
				}
			}
		}

		// Last resort: use original name-based matching
		if !assigned {
			for i, result := range results {
				if strings.Contains(metricName, strings.ToLower(result.Name)) ||
					strings.Contains(strings.ToLower(result.Name), metricName) {
					results[i].Metrics[metricName] = value
					assigned = true
					break
				}
			}
		}

		// No longer create separate entries for unmatched metrics
	}

	// Create summary
	summary := BenchSummary{
		Timestamp:  time.Now().Format(time.RFC3339),
		CommitID:   commitID,
		Branch:     branch,
		GoVersion:  goVersion,
		SystemInfo: systemInfo,
		Results:    results,
	}

	// Output JSON
	jsonData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		fmt.Printf("Error converting to JSON: %v\n", err)
		os.Exit(1)
	}

	// Determine output path
	outputPath := strings.TrimSuffix(inputFile, ".txt") + ".json"

	err = os.WriteFile(outputPath, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing JSON file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("JSON benchmark results written to %s\n", outputPath)
}

// extractMetricsFromSummary parses a benchmark summary section and extracts metrics.
func extractMetricsFromSummary(summary string) map[string]float64 {
	metrics := make(map[string]float64)

	// Common patterns in summaries
	patterns := []struct {
		regex    *regexp.Regexp
		key      string
		valueIdx int
	}{
		{regexp.MustCompile(`Setup time: (.+)`), "setup_time_ns", 1},
		{regexp.MustCompile(`Insertion rate: ([\d,.]+) keys/sec`), "insertion_rate", 1},
		{regexp.MustCompile(`Sequential lookup rate: ([\d,.]+) keys/sec`), "sequential_lookup_rate", 1},
		{regexp.MustCompile(`Random lookup rate: ([\d,.]+) lookups/sec`), "random_lookup_rate", 1},
		{regexp.MustCompile(`Storage efficiency: ([\d,.]+) bytes/key`), "bytes_per_key", 1},
		{regexp.MustCompile(`Total file size: ([\d,.]+) MB`), "file_size_mb", 1},
		{regexp.MustCompile(`Time to insert \d+ keys: .+ \(([\d,.]+) keys/sec\)`), "max_insertion_rate", 1},
		{regexp.MustCompile(`Time to perform \d+ .+ lookups: .+ \(([\d,.]+) lookups/sec\)`), "max_lookup_rate", 1},
	}

	for _, pattern := range patterns {
		if matches := pattern.regex.FindStringSubmatch(summary); len(matches) > pattern.valueIdx {
			// Convert value to float, removing commas
			strValue := strings.ReplaceAll(matches[pattern.valueIdx], ",", "")
			if value, err := strconv.ParseFloat(strValue, 64); err == nil {
				metrics[pattern.key] = value
			}
		}
	}

	return metrics
}

// extractMetricsFromRawOutput finds all metrics mentioned in the raw benchmark output.
func extractMetricsFromRawOutput(content string) map[string]float64 {
	metrics := make(map[string]float64)

	// Look for common metric patterns in the output
	patterns := []struct {
		regex    *regexp.Regexp
		nameIdx  int
		valueIdx int
		prefix   string
	}{
		// Insertion and retrieval rates
		{regexp.MustCompile(`Time to insert (\d+) keys: [^(]+ \(([\d,.]+) keys/sec\)`), 1, 2, "insertion_rate_"},
		{regexp.MustCompile(`Time to insert \d+ keys: [^(]+ \(([\d,.]+) keys/sec\)`), 0, 1, "insertion_rate"},
		{regexp.MustCompile(`Time to verify (\d+) keys: [^(]+ \(([\d,.]+) keys/sec\)`), 1, 2, "verification_rate_"},
		{regexp.MustCompile(`Time to verify \d+ (?:sampled )?keys: [^(]+ \(([\d,.]+) keys/sec\)`), 0, 1, "verification_rate"},
		{regexp.MustCompile(`Time to perform \d+ random lookups: [^(]+ \(([\d,.]+) lookups/sec\)`), 0, 1, "random_lookup_rate"},
		{regexp.MustCompile(`Time to retrieve \d+ UUID keys[^(]+ \(([\d,.]+) keys/sec\)`), 0, 1, "retrieval_rate"},
		{regexp.MustCompile(`Time to validate \d+ UUID keys: [^(]+ \(([\d,.]+) keys/sec\)`), 0, 1, "validation_rate"},

		// Progress reporting
		{regexp.MustCompile(`Time to (\w+) (?:all )?\d+ (\w+)(?:\W+)?\: [^(]+ \(([\d,.]+) \w+/sec\)`), 1, 3, "rate_"},
		{regexp.MustCompile(`(\w+) \d+ (\w+)\.+ \(([\d,.]+) \w+/sec`), 1, 3, "batch_"},

		// File and storage metrics
		{regexp.MustCompile(`File size for (\d+)0000 keys: ([\d,.]+) MB`), 1, 2, "file_size_mb_"},
		{regexp.MustCompile(`Average bytes per (\w+)-(\w+) pair: ([\d,.]+) bytes`), 0, 3, "bytes_per_key"},
		{regexp.MustCompile(`Average bytes per key-value pair: ([\d,.]+) bytes`), 0, 1, "bytes_per_key"},

		// Memory metrics
		{regexp.MustCompile(`Memory: ([\d,.]+)%`), 0, 1, "memory_pct_"},
		{regexp.MustCompile(`Alloc=([\d,.]+)MB`), 0, 1, "memory_alloc_mb_"},
		{regexp.MustCompile(`Sys=([\d,.]+)MB`), 0, 1, "memory_sys_mb_"},
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		for _, pattern := range patterns {
			if matches := pattern.regex.FindStringSubmatch(line); len(matches) > pattern.valueIdx {
				// Generate a metric name based on the context
				var metricName string
				if pattern.nameIdx > 0 && pattern.nameIdx < len(matches) {
					metricName = pattern.prefix + strings.ToLower(matches[pattern.nameIdx])
				} else {
					metricName = pattern.prefix
				}

				// Convert value to float, removing commas
				strValue := strings.ReplaceAll(matches[pattern.valueIdx], ",", "")
				if value, err := strconv.ParseFloat(strValue, 64); err == nil {
					metrics[metricName] = value

					// Also check which benchmark type this metric belongs to
					if strings.Contains(line, "10000000") || strings.Contains(line, "10 million") {
						metrics["TenMillionKeys_"+metricName] = value
					} else if strings.Contains(line, "1000000") || strings.Contains(line, "million") && !strings.Contains(line, "10") {
						metrics["MillionKeys_"+metricName] = value
					} else if strings.Contains(line, "10000") || strings.Contains(line, "thousand") {
						metrics["TenThousandKeys_"+metricName] = value
					} else if strings.Contains(line, "UUID") {
						metrics["UUIDKeys_"+metricName] = value
					}
				}
			}
		}
	}

	return metrics
}
