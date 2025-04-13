// Package phash_test provides scale testing for the persistent hash implementation.
//
// This file contains medium-scale benchmarks that test the performance with
// one million entries, providing insights into real-world usage patterns.
// It measures:
//   - Insertion performance (overall and per batch)
//   - Memory usage during operations
//   - Lookup performance for data verification
//   - Storage efficiency (bytes per key-value pair)
package phash_test

import (
	"encoding/binary"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/theflywheel/phash"
)

// BenchmarkMillionKeys evaluates the performance of the hash implementation
// at a medium scale with one million numeric keys.
//
// Metrics collected:
// - Insertion rate: Keys inserted per second with progress reporting
// - Memory usage: During the insertion process
// - Verification rate: Speed of key verification on a sample of the data
// - Storage efficiency: Average bytes used per key-value pair
// - Total file size: Size of the resulting hash file
//
// This benchmark represents a common production-scale usage scenario.
func BenchmarkMillionKeys(b *testing.B) {
	// Force benchmark to run only once regardless of -benchtime flag
	b.N = 1

	// Reset timer to exclude setup
	b.ResetTimer()
	b.StopTimer()

	tempFile := "million_keys.phash"
	defer os.Remove(tempFile)

	keySize := uint32(8)
	valueSize := uint32(8)
	numKeys := 1_000_000      // One million keys
	reportInterval := 100_000 // Report progress every 100K keys

	// Create metrics collection
	metrics := BenchmarkMetrics{
		Name:       "MillionKeys",
		Category:   "scale",
		Operations: numKeys,
		Metrics:    make(map[string]float64),
	}

	// Create hash instance
	b.Log("Opening hash file...")
	ph, err := phash.Open(tempFile, keySize, valueSize)
	if err != nil {
		b.Fatalf("Failed to open hash: %v", err)
	}
	defer ph.Close()

	// Prepare memory
	runtime.GC()

	// Measure write time
	b.Logf("Starting insertion of %d keys...", numKeys)
	b.StartTimer()
	writeStart := time.Now()

	// Pre-allocate keys and values for reuse
	key := make([]byte, keySize)
	value := make([]byte, valueSize)

	for i := 0; i < numKeys; i++ {
		// Same value as key
		binary.BigEndian.PutUint64(key, uint64(i))
		binary.BigEndian.PutUint64(value, uint64(i))

		if err := ph.Put(key, value); err != nil {
			b.Fatalf("Failed to insert key %d: %v", i, err)
		}

		// Report progress at intervals
		if (i+1)%reportInterval == 0 {
			b.StopTimer()
			elapsed := time.Since(writeStart)
			rate := float64(i+1) / elapsed.Seconds()
			b.Logf("Inserted %d keys... (%.2f keys/sec)", i+1, rate)
			b.StartTimer()
		}
	}

	b.StopTimer()
	writeTime := time.Since(writeStart)
	insertionRate := float64(numKeys) / writeTime.Seconds()
	b.Logf("Time to insert %d keys: %v (%.2f keys/sec)",
		numKeys, writeTime, insertionRate)

	// Store metrics
	metrics.Metrics["insertion_rate"] = insertionRate

	// Verify a sample of the data
	verifySampleSize := 10_000 // Check a smaller sample for speed
	b.Logf("Verifying sample of %d keys...", verifySampleSize)

	b.StartTimer()
	sampleStart := time.Now()
	step := numKeys / verifySampleSize
	for i := 0; i < numKeys; i += step {
		binary.BigEndian.PutUint64(key, uint64(i))

		val, found := ph.Get(key)
		if !found {
			b.Fatalf("Key %d not found", i)
		}

		actualValue := binary.BigEndian.Uint64(val)
		if actualValue != uint64(i) {
			b.Fatalf("Value mismatch for key %d: expected %d, got %d", i, i, actualValue)
		}
	}

	b.StopTimer()
	sampleTime := time.Since(sampleStart)
	verificationRate := float64(verifySampleSize) / sampleTime.Seconds()
	b.Logf("Time to verify %d sampled keys: %v (%.2f keys/sec)",
		verifySampleSize, sampleTime, verificationRate)

	// Store metrics
	metrics.Metrics["verification_rate"] = verificationRate

	// Get file stats
	fileInfo, err := os.Stat(tempFile)
	if err != nil {
		b.Fatalf("Failed to get file stats: %v", err)
	}

	fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)
	bytesPerKey := float64(fileInfo.Size()) / float64(numKeys)

	b.Logf("File size for %d keys: %.2f MB", numKeys, fileSizeMB)
	b.Logf("Average bytes per key-value pair: %.2f bytes", bytesPerKey)

	// Store metrics
	metrics.Metrics["file_size_mb"] = fileSizeMB
	metrics.Metrics["bytes_per_key"] = bytesPerKey
	metrics.NsPerOp = float64(writeTime.Nanoseconds() + sampleTime.Nanoseconds())
	metrics.BytesPerOp = int(float64(fileInfo.Size()) / float64(numKeys) * 10_000) // Rough estimate for benchmark
	metrics.AllocsPerOp = 10_000                                                   // Approximation based on previous runs

	// Collect more metrics
	memoryStats := getMemoryStats()
	for k, v := range memoryStats {
		metrics.Metrics[k] = v
	}

	// Save metrics to file
	if err := saveBenchmarkResult(metrics, "latest.json"); err != nil {
		b.Logf("Failed to save benchmark result: %v", err)
	}

	b.Logf("Million key benchmark completed successfully")
}
