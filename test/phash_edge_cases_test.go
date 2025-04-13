package phash_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/theflywheel/phash"
)

// TestVariousSizes tests different combinations of key and value sizes
func TestVariousSizes(t *testing.T) {
	testCases := []struct {
		name      string
		keySize   uint32
		valueSize uint32
	}{
		{"Small_Keys_Small_Values", 4, 4},      // 4-byte keys, 4-byte values
		{"Small_Keys_Large_Values", 4, 1024},   // 4-byte keys, 1KB values
		{"Large_Keys_Small_Values", 256, 4},    // 256-byte keys, 4-byte values
		{"Large_Keys_Large_Values", 256, 1024}, // 256-byte keys, 1KB values
		{"Equal_Keys_Values", 16, 16},          // Equal key and value sizes
		{"Tiny_Keys_Values", 1, 1},             // Minimum size
		{"Medium_Keys_Values", 32, 64},         // Medium sizes
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempFile := "size_test_" + tc.name + ".phash"
			defer os.Remove(tempFile)

			// Create hash
			ph, err := phash.Open(tempFile, tc.keySize, tc.valueSize)
			if err != nil {
				t.Fatalf("Failed to open hash with key size %d and value size %d: %v",
					tc.keySize, tc.valueSize, err)
			}
			defer ph.Close()

			// Create test key and value of the specified sizes
			key := make([]byte, tc.keySize)
			value := make([]byte, tc.valueSize)

			// Fill with distinct patterns
			for i := range key {
				key[i] = byte(i % 256)
			}
			for i := range value {
				value[i] = byte((i + 128) % 256)
			}

			// Store value
			if err := ph.Put(key, value); err != nil {
				t.Fatalf("Failed to put value: %v", err)
			}

			// Retrieve value
			retrievedValue, found := ph.Get(key)
			if !found {
				t.Fatal("Key not found")
			}

			// Check value
			if !bytes.Equal(retrievedValue, value) {
				t.Errorf("Value mismatch for key size %d and value size %d",
					tc.keySize, tc.valueSize)
			}
		})
	}
}

// TestResizing tests that the hash table correctly resizes as it grows
func TestResizing(t *testing.T) {
	tempFile := "resize_test.phash"
	defer os.Remove(tempFile)

	keySize := uint32(8)
	valueSize := uint32(8)

	// Create new hash
	ph, err := phash.Open(tempFile, keySize, valueSize)
	if err != nil {
		t.Fatalf("Failed to open hash: %v", err)
	}
	defer ph.Close()

	// Insert enough entries to trigger multiple resizes
	numEntries := 5000 // Should trigger at least a couple of resizes

	for i := 0; i < numEntries; i++ {
		key := make([]byte, keySize)
		value := make([]byte, valueSize)

		// Fill key and value
		for j := range key {
			key[j] = byte((i + j) % 256)
		}
		for j := range value {
			value[j] = byte((i + j + 128) % 256)
		}

		if err := ph.Put(key, value); err != nil {
			t.Fatalf("Failed to put entry %d: %v", i, err)
		}

		// Verify the entry was stored correctly
		retrievedValue, found := ph.Get(key)
		if !found {
			t.Fatalf("Entry %d not found immediately after insertion", i)
		}

		if !bytes.Equal(retrievedValue, value) {
			t.Errorf("Value mismatch for entry %d", i)
		}
	}

	// Final verification of a sample of entries
	for i := 0; i < numEntries; i += (numEntries / 100) { // Check ~100 entries
		key := make([]byte, keySize)
		expectedValue := make([]byte, valueSize)

		// Recreate the same key and value patterns
		for j := range key {
			key[j] = byte((i + j) % 256)
		}
		for j := range expectedValue {
			expectedValue[j] = byte((i + j + 128) % 256)
		}

		retrievedValue, found := ph.Get(key)
		if !found {
			t.Fatalf("Entry %d not found after all insertions", i)
		}

		if !bytes.Equal(retrievedValue, expectedValue) {
			t.Errorf("Value mismatch for entry %d after all insertions", i)
		}
	}
}

// TestEmptyValue tests storing and retrieving empty values (zero-length)
func TestEmptyValue(t *testing.T) {
	tempFile := "empty_value_test.phash"
	defer os.Remove(tempFile)

	keySize := uint32(8)
	valueSize := uint32(0) // Zero-length values

	// Create hash
	ph, err := phash.Open(tempFile, keySize, valueSize)
	if err != nil {
		t.Fatalf("Failed to open hash with zero-length values: %v", err)
	}
	defer ph.Close()

	// Create and store a key with an empty value
	key := make([]byte, keySize)
	for i := range key {
		key[i] = byte(i)
	}

	emptyValue := make([]byte, 0)

	// Store the empty value
	if err := ph.Put(key, emptyValue); err != nil {
		t.Fatalf("Failed to store empty value: %v", err)
	}

	// Retrieve the value
	retrievedValue, found := ph.Get(key)
	if !found {
		t.Fatal("Key with empty value not found")
	}

	// Check that the retrieved value is indeed empty
	if len(retrievedValue) != 0 {
		t.Errorf("Expected empty value, got value of length %d", len(retrievedValue))
	}
}
