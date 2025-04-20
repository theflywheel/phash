package phash_test

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"

	"github.com/theflywheel/phash"
)

func TestBasicOperations(t *testing.T) {
	tempFile := "basic_test.phash"
	defer os.Remove(tempFile)

	keySize := uint32(8)
	valueSize := uint32(8)

	ph, err := phash.Open(tempFile, keySize, valueSize)
	if err != nil {
		t.Fatalf("Failed to open hash: %v", err)
	}
	defer ph.Close()

	for i := uint64(0); i < 10; i++ {
		key := make([]byte, keySize)
		value := make([]byte, valueSize)

		binary.BigEndian.PutUint64(key, i)
		binary.BigEndian.PutUint64(value, i*100)

		if err := ph.Put(key, value); err != nil {
			t.Fatalf("Failed to put key %d: %v", i, err)
		}
	}

	for i := uint64(0); i < 10; i++ {
		key := make([]byte, keySize)
		binary.BigEndian.PutUint64(key, i)

		expectedValue := make([]byte, valueSize)
		binary.BigEndian.PutUint64(expectedValue, i*100)

		value, found := ph.Get(key)
		if !found {
			t.Fatalf("Key %d not found", i)
		}

		if !bytes.Equal(value, expectedValue) {
			t.Errorf("Value mismatch for key %d: expected %v, got %v",
				i, expectedValue, value)
		}
	}
}

func TestPersistence(t *testing.T) {
	tempFile := "persistence_test.phash"
	defer os.Remove(tempFile)

	keySize := uint32(8)
	valueSize := uint32(8)

	{
		ph, err := phash.Open(tempFile, keySize, valueSize)
		if err != nil {
			t.Fatalf("Failed to open hash: %v", err)
		}

		for i := uint64(0); i < 10; i++ {
			key := make([]byte, keySize)
			value := make([]byte, valueSize)

			binary.BigEndian.PutUint64(key, i)
			binary.BigEndian.PutUint64(value, i*100)

			if err := ph.Put(key, value); err != nil {
				t.Fatalf("Failed to put key %d: %v", i, err)
			}
		}

		if err := ph.Close(); err != nil {
			t.Fatalf("Failed to close hash: %v", err)
		}
	}

	{
		ph2, err := phash.Open(tempFile, keySize, valueSize)
		if err != nil {
			t.Fatalf("Failed to reopen hash: %v", err)
		}
		defer ph2.Close()

		for i := uint64(0); i < 10; i++ {
			key := make([]byte, keySize)
			binary.BigEndian.PutUint64(key, i)

			expectedValue := make([]byte, valueSize)
			binary.BigEndian.PutUint64(expectedValue, i*100)

			value, found := ph2.Get(key)
			if !found {
				t.Fatalf("Key %d not found after reopen", i)
			}

			if !bytes.Equal(value, expectedValue) {
				t.Errorf("Value mismatch for key %d after reopen: expected %v, got %v",
					i, expectedValue, value)
			}
		}
	}
}

func TestInvalidInputs(t *testing.T) {
	tempFile := "invalid_test.phash"
	defer os.Remove(tempFile)

	keySize := uint32(8)
	valueSize := uint32(8)

	ph, err := phash.Open(tempFile, keySize, valueSize)
	if err != nil {
		t.Fatalf("Failed to open hash: %v", err)
	}
	defer ph.Close()

	// Test invalid key size
	invalidKey := make([]byte, keySize-1) // Too small
	value := make([]byte, valueSize)

	err = ph.Put(invalidKey, value)
	if err == nil {
		t.Error("Expected error for invalid key size, got nil")
	}

	// Test invalid value size
	key := make([]byte, keySize)
	invalidValue := make([]byte, valueSize+1) // Too large

	err = ph.Put(key, invalidValue)
	if err == nil {
		t.Error("Expected error for invalid value size, got nil")
	}

	// Test Get with invalid key size
	_, found := ph.Get(invalidKey)
	if found {
		t.Error("Expected key not found for invalid key size")
	}
}

// TestOverwrite tests overwriting existing keys
func TestOverwrite(t *testing.T) {
	tempFile := "overwrite_test.phash"
	defer os.Remove(tempFile)

	keySize := uint32(8)
	valueSize := uint32(8)

	ph, err := phash.Open(tempFile, keySize, valueSize)
	if err != nil {
		t.Fatalf("Failed to open hash: %v", err)
	}
	defer ph.Close()

	// Insert an entry
	key := make([]byte, keySize)
	value1 := make([]byte, valueSize)
	binary.BigEndian.PutUint64(key, 42)
	binary.BigEndian.PutUint64(value1, 100)

	if err := ph.Put(key, value1); err != nil {
		t.Fatalf("Failed to put initial value: %v", err)
	}

	// Verify the entry
	result, found := ph.Get(key)
	if !found {
		t.Fatal("Key not found")
	}

	initialValue := binary.BigEndian.Uint64(result)
	if initialValue != 100 {
		t.Fatalf("Expected value 100, got %d", initialValue)
	}

	// Overwrite the entry
	value2 := make([]byte, valueSize)
	binary.BigEndian.PutUint64(value2, 200)

	if err := ph.Put(key, value2); err != nil {
		t.Fatalf("Failed to overwrite value: %v", err)
	}

	// Verify the overwritten entry
	result, found = ph.Get(key)
	if !found {
		t.Fatal("Key not found after overwrite")
	}

	newValue := binary.BigEndian.Uint64(result)
	if newValue != 200 {
		t.Fatalf("Expected updated value 200, got %d", newValue)
	}
}
