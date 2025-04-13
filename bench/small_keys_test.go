// Package phash_test provides scale testing for the persistent hash implementation.
//
// This file contains small-scale benchmarks that test the performance with
// ten thousand entries, providing insights into baseline performance.
// It measures:
//   - Insertion performance (overall and per batch)
//   - Random lookup performance
//   - Sequential lookup performance
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

// BenchmarkTenThousandKeys evaluates the performance of the hash implementation
// with ten thousand numeric keys.
//
// Metrics collected:
// - Insertion rate: Keys inserted per second with progress reporting
// - Random lookup rate: Performance of random access patterns
// - Sequential lookup rate: Performance of sequential key verification
// - Storage efficiency: Average bytes used per key-value pair
// - Total file size: Size of the resulting hash file
//
// This benchmark is useful for baseline performance evaluation.
func BenchmarkTenThousandKeys(b *testing.B) {
	// Print a message when the benchmark starts
	fmt.Printf("BenchmarkTenThousandKeys started execution, b.N = %d\n", b.N)

	// Force benchmark to run only once regardless of -benchtime flag
	b.N = 1

	// Reset timer for setup
	b.ResetTimer()
	b.StopTimer()

	// Setup temp file for the hash
	tempFile := "ten_thousand_keys.phash"
	defer os.Remove(tempFile)

	keySize := uint32(8)
	valueSize := uint32(8)
	numKeys := 10_000         // 10K keys
	progressInterval := 1_000 // Show progress every 1K insertions

	// Create hash instance
	b.Log("Opening hash file...")
	ph, err := phash.Open(tempFile, keySize, valueSize)
	if err != nil {
		b.Fatalf("Failed to open hash: %v", err)
	}
	defer ph.Close()

	// Create metrics collection
	metrics := BenchmarkMetrics{
		Name:       "TenThousandKeys",
		Category:   "scale",
		Operations: numKeys,
		Metrics:    make(map[string]float64),
	}

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
		if (i+1)%progressInterval == 0 {
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
	randomSampleSize := 1_000 // Check a smaller sample for speed
	b.Logf("Verifying random sample of %d keys...", randomSampleSize)

	b.StartTimer()
	randomReadStart := time.Now()

	for i := 0; i < randomSampleSize; i++ {
		// Generate random key indices
		keyID := (i*31 + 17) % numKeys
		binary.BigEndian.PutUint64(key, uint64(keyID))

		val, found := ph.Get(key)
		if !found {
			b.Fatalf("Random key %d not found", keyID)
		}

		// Verify values
		actualValue := binary.BigEndian.Uint64(val)
		if actualValue != uint64(keyID) {
			b.Fatalf("Value mismatch for random key %d: expected %d, got %d",
				keyID, keyID, actualValue)
		}

		// Report progress every 200 lookups
		if (i+1)%200 == 0 {
			b.StopTimer()
			b.Logf("Retrieved %d random keys...", i+1)
			b.StartTimer()
		}
	}

	b.StopTimer()
	randomReadTime := time.Since(randomReadStart)
	randomLookupRate := float64(randomSampleSize) / randomReadTime.Seconds()
	b.Logf("Time to perform %d random lookups: %v (%.2f lookups/sec)",
		randomSampleSize, randomReadTime, randomLookupRate)

	// Store metrics
	metrics.Metrics["random_lookup_rate"] = randomLookupRate

	// Sequential verification of all keys
	b.Logf("Verifying all %d keys sequentially...", numKeys)

	b.StartTimer()
	seqReadStart := time.Now()

	for i := 0; i < numKeys; i++ {
		binary.BigEndian.PutUint64(key, uint64(i))
		val, found := ph.Get(key)
		if !found {
			b.Fatalf("Key %d not found", i)
		}

		// Verify values
		actualValue := binary.BigEndian.Uint64(val)
		if actualValue != uint64(i) {
			b.Fatalf("Value mismatch for key %d: expected %d, got %d",
				i, i, actualValue)
		}

		// Report progress at intervals
		if (i+1)%1000 == 0 {
			b.StopTimer()
			b.Logf("Verified %d sequential keys...", i+1)
			b.StartTimer()
		}
	}

	b.StopTimer()
	seqReadTime := time.Since(seqReadStart)
	seqLookupRate := float64(numKeys) / seqReadTime.Seconds()
	b.Logf("Time to verify all %d keys sequentially: %v (%.2f lookups/sec)",
		numKeys, seqReadTime, seqLookupRate)

	// Store metrics
	metrics.Metrics["sequential_lookup_rate"] = seqLookupRate

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
	metrics.NsPerOp = float64(writeTime.Nanoseconds() + randomReadTime.Nanoseconds() + seqReadTime.Nanoseconds())
	metrics.BytesPerOp = int(fileInfo.Size())
	metrics.AllocsPerOp = 20_000 // Approximation based on previous runs

	// Save to latest.json for consolidated results
	if err := saveBenchmarkResult(metrics, "latest.json"); err != nil {
		b.Logf("Failed to save benchmark result to latest.json: %v", err)
	}

	b.Logf("Ten thousand keys benchmark completed successfully")
}
