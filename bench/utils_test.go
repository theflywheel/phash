package phash_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// BenchmarkMetrics represents metrics for a single benchmark
type BenchmarkMetrics struct {
	Name        string             `json:"name"`
	Category    string             `json:"category"`
	Operations  int                `json:"operations"`
	NsPerOp     float64            `json:"ns_per_op"`
	BytesPerOp  int                `json:"bytes_per_op,omitempty"`
	AllocsPerOp int                `json:"allocs_per_op,omitempty"`
	Metrics     map[string]float64 `json:"metrics"`
}

// BenchmarkSummary represents all benchmark results
type BenchmarkSummary struct {
	Timestamp string             `json:"timestamp"`
	CommitID  string             `json:"commit_id"`
	Branch    string             `json:"branch"`
	GoVersion string             `json:"go_version"`
	Results   []BenchmarkMetrics `json:"results"`
}

// getMemoryUsage returns the current memory stats as a formatted string
func getMemoryUsage() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return fmt.Sprintf("Memory: Alloc=%.1fMB Sys=%.1fMB",
		float64(m.Alloc)/1024/1024,
		float64(m.Sys)/1024/1024)
}

// getMemoryStats returns the current memory stats as a map
func getMemoryStats() map[string]float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return map[string]float64{
		"alloc_mb": float64(m.Alloc) / (1024 * 1024),
		"sys_mb":   float64(m.Sys) / (1024 * 1024),
	}
}

// cleanupMetrics removes unwanted detailed metrics like batch_rate_* or memory_mb_*
func cleanupMetrics(metrics *BenchmarkMetrics) {
	if metrics.Metrics == nil {
		return
	}

	filteredMetrics := make(map[string]float64)
	for key, value := range metrics.Metrics {
		// Skip metrics that match unwanted patterns
		if strings.HasPrefix(key, "batch_rate_") ||
			strings.HasPrefix(key, "memory_mb_") ||
			strings.HasPrefix(key, "batch_insert_") ||
			strings.HasPrefix(key, "batch_retrieve_") ||
			strings.HasPrefix(key, "batch_validate_") {
			continue
		}
		filteredMetrics[key] = value
	}

	metrics.Metrics = filteredMetrics
}

// saveBenchmarkResult saves a benchmark result to the benchmark_history directory
func saveBenchmarkResult(metrics BenchmarkMetrics, resultsFile string) error {
	// Clean up metrics before saving
	cleanupMetrics(&metrics)

	// Determine the absolute path of the repository root (assume we're in a subdirectory)
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
	}

	// Get the repository root by going up one level (from bench to repo root)
	repoRoot := filepath.Dir(currentDir)

	// Create benchmark_history directory in the repository root
	benchmarkDir := filepath.Join(repoRoot, "benchmark_history")
	err = os.MkdirAll(benchmarkDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Get git info if available
	commitID := "local"
	branch := "dev"

	// Try to get git info from the repository root
	gitHeadPath := filepath.Join(repoRoot, ".git", "HEAD")
	if gitHead, err := os.ReadFile(gitHeadPath); err == nil {
		headContent := string(gitHead)
		if len(headContent) > 0 {
			// For branches it looks like "ref: refs/heads/main"
			if strings.HasPrefix(headContent, "ref: refs/heads/") {
				branch = strings.TrimPrefix(headContent, "ref: refs/heads/")
				branch = strings.TrimSpace(branch)
			}

			// Try to get commit ID
			refPath := strings.TrimPrefix(strings.TrimSpace(headContent), "ref: ")
			refFile := filepath.Join(repoRoot, ".git", refPath)
			if _, err := os.Stat(refFile); err == nil {
				if commitData, err := os.ReadFile(refFile); err == nil {
					commitID = strings.TrimSpace(string(commitData))
					if len(commitID) >= 8 {
						commitID = commitID[:8] // First 8 chars
					}
				}
			}
		}
	}

	// Create summary object
	summary := BenchmarkSummary{
		Timestamp: time.Now().Format(time.RFC3339),
		CommitID:  commitID,
		Branch:    branch,
		GoVersion: runtime.Version(),
		Results:   []BenchmarkMetrics{metrics},
	}

	// Merge with existing results if available
	latestFile := filepath.Join(benchmarkDir, resultsFile)
	existingData, err := os.ReadFile(latestFile)
	if err == nil {
		var existingSummary BenchmarkSummary
		if err := json.Unmarshal(existingData, &existingSummary); err == nil {
			// Keep same timestamp and git info, just append result
			summary.Results = append(existingSummary.Results, metrics)
		}
	}

	// Write to output file
	jsonData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	// Write only to the specified file
	err = os.WriteFile(latestFile, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	fmt.Printf("Benchmark results saved to: %s\n", latestFile)

	return nil
}
