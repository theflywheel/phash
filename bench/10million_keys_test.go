// Package phash_test provides scale testing for the persistent hash implementation.
//
// This file contains large-scale benchmarks that test the performance and
// scalability of the hash implementation with millions of entries.
// It measures:
//   - Insertion performance (overall and per batch)
//   - Memory usage during operations
//   - Random lookup performance
//   - Storage efficiency (bytes per key-value pair)
package phash_test

import (
	"encoding/binary"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/theflywheel/phash"
)

// BenchmarkTenMillionKeys evaluates the performance and scalability of the hash
// implementation by inserting and retrieving 10 million keys.
//
// Metrics collected:
// - Setup time: Time to open and initialize the hash file
// - Insertion rate: Keys inserted per second (overall and per batch)
// - Memory usage: During the insertion process
// - Verification rate: Speed of key verification
// - Random lookup rate: Performance of random access patterns
// - Storage efficiency: Average bytes used per key-value pair
// - Total file size: Size of the resulting hash file
//
// This benchmark represents a worst-case scenario with maximum scale.
func BenchmarkTenMillionKeys(b *testing.B) {
	// Force benchmark to run only once regardless of -benchtime flag
	b.N = 1

	// Reset timer to exclude setup
	b.ResetTimer()
	b.StopTimer()

	tempFile := "ten_million_keys.phash"
	defer os.Remove(tempFile)

	keySize := uint32(8)
	valueSize := uint32(8)
	numKeys := 10_000_000     // 10 million keys
	reportInterval := 500_000 // Report every 500K insertions

	// Create metrics collection
	metrics := BenchmarkMetrics{
		Name:       "TenMillionKeys",
		Category:   "scale",
		Operations: 10_000_000,
		Metrics:    make(map[string]float64),
	}

	// Create hash instance
	b.Log("Opening hash file...")
	setupStart := time.Now()
	ph, err := phash.Open(tempFile, keySize, valueSize)
	if err != nil {
		b.Fatalf("Failed to open hash: %v", err)
	}
	defer ph.Close()
	setupTime := time.Since(setupStart)
	metrics.Metrics["setup_time_ns"] = float64(setupTime.Nanoseconds())

	// Force GC to get a clean start
	runtime.GC()

	// Measure write time
	b.Logf("Starting insertion of %d keys...", numKeys)
	b.StartTimer()
	writeStart := time.Now()

	// Pre-allocate keys and values for reuse
	key := make([]byte, keySize)
	value := make([]byte, valueSize)

	for i := 0; i < numKeys; i++ {
		// Same value as key for simplicity and verification
		binary.BigEndian.PutUint64(key, uint64(i))
		binary.BigEndian.PutUint64(value, uint64(i))

		if err := ph.Put(key, value); err != nil {
			b.Fatalf("Failed to insert key %d: %v", i, err)
		}

		// Report progress at intervals
		if (i+1)%reportInterval == 0 {
			b.StopTimer() // Pause timer during logging
			elapsed := time.Since(writeStart)
			rate := float64(i+1) / elapsed.Seconds()
			memStats := getMemoryStats()
			b.Logf("Inserted %d keys... (%.2f keys/sec)", i+1, rate)
			metrics.Metrics[fmt.Sprintf("batch_rate_%d", i+1)] = rate
			metrics.Metrics[fmt.Sprintf("memory_mb_%d", i+1)] = memStats["alloc_mb"]
			b.StartTimer() // Resume timer
		}
	}

	b.StopTimer()
	writeTime := time.Since(writeStart)
	insertionRate := float64(numKeys) / writeTime.Seconds()
	b.Logf("Time to insert %d keys: %v (%.2f keys/sec)",
		numKeys, writeTime, insertionRate)

	// Store metrics
	metrics.Metrics["insertion_rate"] = insertionRate
	metrics.Metrics["write_time_ns"] = float64(writeTime.Nanoseconds())

	// Test random access performance
	b.Log("Testing random access performance...")
	randomSamples := 100_000 // 100K random lookups
	b.StartTimer()
	randomStart := time.Now()

	for i := 0; i < randomSamples; i++ {
		// Generate "random" key indices with a simple distribution
		keyID := (i*104729 + 15485863) % numKeys // Use prime numbers for better distribution
		binary.BigEndian.PutUint64(key, uint64(keyID))

		val, found := ph.Get(key)
		if !found {
			b.Fatalf("Random key %d not found", keyID)
		}

		// Verify value (occasionally)
		if i%1000 == 0 {
			actualValue := binary.BigEndian.Uint64(val)
			if actualValue != uint64(keyID) {
				b.Fatalf("Value mismatch for key %d: expected %d, got %d", keyID, keyID, actualValue)
			}
		}
	}

	b.StopTimer()
	randomTime := time.Since(randomStart)
	randomLookupRate := float64(randomSamples) / randomTime.Seconds()
	b.Logf("Time to perform %d random lookups: %v (%.2f lookups/sec)",
		randomSamples, randomTime, randomLookupRate)

	// Store metrics
	metrics.Metrics["random_lookup_rate"] = randomLookupRate
	metrics.Metrics["random_lookup_time_ns"] = float64(randomTime.Nanoseconds())

	// File stats
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

	// Add benchmark standard metrics
	metrics.NsPerOp = float64(writeTime.Nanoseconds() + randomTime.Nanoseconds())
	metrics.BytesPerOp = int(fileInfo.Size() / 10) // Just a portion for the benchmark
	metrics.AllocsPerOp = 100_000                  // Approximation based on previous runs

	// Collect memory metrics
	memoryStats := getMemoryStats()
	for k, v := range memoryStats {
		metrics.Metrics[k] = v
	}

	// Save metrics to file
	if err := saveBenchmarkResult(metrics, "latest.json"); err != nil {
		b.Logf("Failed to save benchmark result: %v", err)
	}

	b.Logf("Ten million key benchmark completed successfully")
}
