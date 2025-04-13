package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/theflywheel/phash"
)

func main() {
	// Clean up previous example
	os.Remove("example.phash")

	// Open or create a persistent hash
	ph, err := phash.Open("example.phash", 8, 8) // 8-byte keys and values
	if err != nil {
		log.Fatalf("Failed to open hash: %v", err)
	}
	defer ph.Close()

	fmt.Println("Persistent hash opened successfully")

	// Insert some data
	for i := 0; i < 10; i++ {
		key := make([]byte, 8)
		value := make([]byte, 8)

		binary.BigEndian.PutUint64(key, uint64(i))
		binary.BigEndian.PutUint64(value, uint64(i*100))

		err = ph.Put(key, value)
		if err != nil {
			log.Fatalf("Failed to insert key %d: %v", i, err)
		}
	}

	fmt.Println("Inserted 10 key-value pairs")

	// Retrieve and display some values
	for i := 0; i < 15; i += 2 {
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, uint64(i))

		value, found := ph.Get(key)
		if found {
			val := binary.BigEndian.Uint64(value)
			fmt.Printf("Key %d => Value %d\n", i, val)
		} else {
			fmt.Printf("Key %d not found\n", i)
		}
	}

	// Update a value
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, uint64(2))

	newValue := make([]byte, 8)
	binary.BigEndian.PutUint64(newValue, uint64(999))

	err = ph.Put(key, newValue)
	if err != nil {
		log.Fatalf("Failed to update key: %v", err)
	}

	// Verify the update
	value, found := ph.Get(key)
	if found {
		val := binary.BigEndian.Uint64(value)
		fmt.Printf("Updated key 2 => Value %d\n", val)
	}

	fmt.Println("Example completed successfully")
}
