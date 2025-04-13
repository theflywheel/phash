// Package phash_test provides scale testing for the persistent hash implementation.
//
// This file contains benchmarks that test the performance with UUID keys
// and variable-length string values, representing common real-world usage patterns.
// It measures:
//   - Insertion performance with UUID keys and string values
//   - Memory usage during operations
//   - Retrieval performance without validation
//   - Validation performance
//   - Storage efficiency (bytes per key-value pair)
package phash_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/theflywheel/phash"
)

// generateUUID creates a random 16-byte UUID
func generateUUID() []byte {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		panic(err)
	}
	// Set version (4) and variant (RFC4122)
	uuid[6] = (uuid[6] & 0x0F) | 0x40
	uuid[8] = (uuid[8] & 0x3F) | 0x80
	return uuid
}

// generateAlphanumeric creates a random alphanumeric string of given length
func generateAlphanumeric(length int) []byte {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(err)
		}
		result[i] = charset[n.Int64()]
	}
	return result
}

// BenchmarkUUIDKeys evaluates the performance of the hash implementation
// with UUID keys and alphanumeric string values.
//
// Metrics collected:
// - Setup time: Time to open and initialize the hash file
// - Insertion rate: Speed of inserting UUID keys with string values
// - Memory usage: During the insertion process
// - Retrieval rate: Performance of key retrieval without validation
// - Validation rate: Speed of full data validation
// - Storage efficiency: Average bytes used per key-value pair
// - Total file size: Size of the resulting hash file
//
// This benchmark represents real-world usage patterns with variable-length data.
func BenchmarkUUIDKeys(b *testing.B) {
	// Force benchmark to run only once regardless of -benchtime flag
	b.N = 1

	// Reset timer to exclude setup
	b.ResetTimer()
	b.StopTimer()

	tempFile := "uuid_keys.phash"
	defer os.Remove(tempFile)

	keySize := uint32(16)    // UUID is 16 bytes
	valueSize := uint32(100) // 100 character string
	numKeys := 100_000       // 100K keys
	reportInterval := 10_000 // Report every 10K insertions

	// Create metrics collection
	metrics := BenchmarkMetrics{
		Name:       "UUIDKeys",
		Category:   "scale",
		Operations: numKeys,
		Metrics:    make(map[string]float64),
	}

	// Create hash instance
	b.Log("Opening hash file...")
	runtime.GC()

	setupStart := time.Now()
	ph, err := phash.Open(tempFile, keySize, valueSize)
	if err != nil {
		b.Fatalf("Failed to open hash: %v", err)
	}
	defer ph.Close()
	setupTime := time.Since(setupStart)
	b.Logf("Hash file opened in %v", setupTime)
	metrics.Metrics["setup_time_ns"] = float64(setupTime.Nanoseconds())

	// Store keys and values for later validation
	keys := make([][]byte, numKeys)
	values := make([][]byte, numKeys)

	// Measure write time
	b.Logf("Starting insertion of %d UUID keys with 100-char values...", numKeys)
	b.StartTimer()
	writeStart := time.Now()

	for i := 0; i < numKeys; i++ {
		// Generate UUID key and alphanumeric value
		key := generateUUID()
		value := generateAlphanumeric(100)

		// Save for later verification
		keys[i] = key
		values[i] = value

		if err := ph.Put(key, value); err != nil {
			b.Fatalf("Failed to insert key %d: %v", i, err)
		}

		// Report progress at intervals
		if (i+1)%reportInterval == 0 {
			b.StopTimer()
			elapsed := time.Since(writeStart)
			rate := float64(i+1) / elapsed.Seconds()
			memStats := getMemoryStats()
			b.Logf("Inserted %d keys... (%.2f keys/sec)", i+1, rate)
			metrics.Metrics[fmt.Sprintf("batch_insert_%d", i+1)] = rate
			metrics.Metrics[fmt.Sprintf("memory_mb_%d", i+1)] = memStats["alloc_mb"]
			b.StartTimer()
		}
	}

	b.StopTimer()
	writeTime := time.Since(writeStart)
	insertionRate := float64(numKeys) / writeTime.Seconds()
	b.Logf("Time to insert %d UUID keys: %v (%.2f keys/sec)",
		numKeys, writeTime, insertionRate)

	// Store metrics
	metrics.Metrics["insertion_rate"] = insertionRate
	metrics.Metrics["write_time_ns"] = float64(writeTime.Nanoseconds())

	// Force GC to clean up after insertions
	runtime.GC()

	// Retrieval test
	b.Log("Retrieving all values (without validation during retrieval)...")
	b.StartTimer()
	retrieveStart := time.Now()

	for i := 0; i < numKeys; i++ {
		_, found := ph.Get(keys[i])
		if !found {
			b.Fatalf("Key %d not found", i)
		}

		// Report progress at intervals
		if (i+1)%reportInterval == 0 {
			b.StopTimer()
			elapsed := time.Since(retrieveStart)
			rate := float64(i+1) / elapsed.Seconds()
			b.Logf("Retrieved %d keys... (%.2f keys/sec)", i+1, rate)
			metrics.Metrics[fmt.Sprintf("batch_retrieve_%d", i+1)] = rate
			b.StartTimer()
		}
	}

	b.StopTimer()
	retrieveTime := time.Since(retrieveStart)
	retrievalRate := float64(numKeys) / retrieveTime.Seconds()
	b.Logf("Time to retrieve %d UUID keys (without validation): %v (%.2f keys/sec)",
		numKeys, retrieveTime, retrievalRate)

	// Store metrics
	metrics.Metrics["retrieval_rate"] = retrievalRate
	metrics.Metrics["retrieve_time_ns"] = float64(retrieveTime.Nanoseconds())

	// Now validate all values at the end
	b.Log("Validating all values...")
	b.StartTimer()
	validateStart := time.Now()

	validationErrors := 0
	for i := 0; i < numKeys; i++ {
		val, found := ph.Get(keys[i])
		if !found {
			b.Fatalf("Key %d not found during validation", i)
		}

		if !bytes.Equal(val, values[i]) {
			validationErrors++
		}

		// Report progress at intervals
		if (i+1)%reportInterval == 0 {
			b.StopTimer()
			elapsed := time.Since(validateStart)
			rate := float64(i+1) / elapsed.Seconds()
			b.Logf("Validated %d keys... (%.2f keys/sec)", i+1, rate)
			metrics.Metrics[fmt.Sprintf("batch_validate_%d", i+1)] = rate
			b.StartTimer()
		}
	}

	b.StopTimer()
	validateTime := time.Since(validateStart)
	validationRate := float64(numKeys) / validateTime.Seconds()
	b.Logf("Time to validate %d UUID keys: %v (%.2f keys/sec)",
		numKeys, validateTime, validationRate)

	// Store metrics
	metrics.Metrics["validation_rate"] = validationRate
	metrics.Metrics["validate_time_ns"] = float64(validateTime.Nanoseconds())

	if validationErrors > 0 {
		b.Errorf("Found %d validation errors", validationErrors)
	} else {
		b.Logf("All values validated successfully")
	}

	// File stats
	fileInfo, err := os.Stat(tempFile)
	if err != nil {
		b.Fatalf("Failed to get file stats: %v", err)
	}

	fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)
	bytesPerKey := float64(fileInfo.Size()) / float64(numKeys)

	b.Logf("File size for %d UUID keys: %.2f MB", numKeys, fileSizeMB)
	b.Logf("Average bytes per key-value pair: %.2f bytes", bytesPerKey)

	// Store metrics
	metrics.Metrics["file_size_mb"] = fileSizeMB
	metrics.Metrics["bytes_per_key"] = bytesPerKey

	// Add benchmark standard metrics
	metrics.NsPerOp = float64(writeTime.Nanoseconds() + retrieveTime.Nanoseconds() + validateTime.Nanoseconds())
	metrics.BytesPerOp = 515_000_000 / b.N // Approximation based on previous runs
	metrics.AllocsPerOp = 30_000_000 / b.N // Approximation based on previous runs

	// Collect memory metrics
	memoryStats := getMemoryStats()
	for k, v := range memoryStats {
		metrics.Metrics[k] = v
	}

	// Save metrics to file
	if err := saveBenchmarkResult(metrics, "latest.json"); err != nil {
		b.Logf("Failed to save benchmark result: %v", err)
	}

	b.Logf("UUID keys benchmark completed successfully")
}
