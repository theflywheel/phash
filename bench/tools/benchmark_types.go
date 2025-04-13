package main

import (
	"time"
)

// BenchmarkResult represents a single benchmark result
type BenchmarkResult struct {
	Name        string  `json:"name"`
	Operations  int     `json:"operations"`
	NsPerOp     float64 `json:"ns_per_op"`
	BytesPerOp  int     `json:"bytes_per_op,omitempty"`
	AllocsPerOp int     `json:"allocs_per_op,omitempty"`
}

// BenchmarkSummary represents the complete benchmark output
type BenchmarkSummary struct {
	Timestamp string            `json:"timestamp"`
	CommitID  string            `json:"commit_id"`
	Branch    string            `json:"branch"`
	GoVersion string            `json:"go_version"`
	Results   []BenchmarkResult `json:"results"`
}

// CreateTimestamp returns a formatted timestamp for benchmark files
func CreateTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

// Comparison represents a comparison between two benchmark results
type Comparison struct {
	Name            string  `json:"name"`
	BaseNsPerOp     float64 `json:"base_ns_per_op"`
	CurrentNsPerOp  float64 `json:"current_ns_per_op"`
	PercentChange   float64 `json:"percent_change"`
	BytesChange     float64 `json:"bytes_change,omitempty"`
	AllocsChange    float64 `json:"allocs_change,omitempty"`
	HasBytesChange  bool    `json:"has_bytes_change,omitempty"`
	HasAllocsChange bool    `json:"has_allocs_change,omitempty"`
	IsRegression    bool    `json:"is_regression"`
}
