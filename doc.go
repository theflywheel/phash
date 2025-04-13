/*
Package phash provides a persistent hash table implementation using memory-mapped files.

PersistentHash is designed to be a high-performance key-value store that persists
data to disk while maintaining fast in-memory access speeds. It uses memory mapping
to provide direct access to the data without copying it into user space.

Basic usage:

	import "github.com/theflywheel/phash"

	// Open or create a persistent hash
	ph, err := phash.Open("data.phash", 8, 8) // 8-byte keys and values
	if err != nil {
		log.Fatal(err)
	}
	defer ph.Close()

	// Insert data
	key := make([]byte, 8)
	value := make([]byte, 8)
	binary.BigEndian.PutUint64(key, 12345)
	binary.BigEndian.PutUint64(value, 67890)
	err = ph.Put(key, value)

	// Retrieve data
	result, ok := ph.Get(key)
	if ok {
		val := binary.BigEndian.Uint64(result)
		fmt.Println("Value:", val)
	}

Features:

  - Fixed-size keys and values for optimal performance
  - Memory-mapped file storage for persistence and fast access
  - Thread-safe with read/write mutex
  - Automatic resizing when load factor exceeds 0.7
  - Uses FNV-1a hashing algorithm for good distribution
  - Open addressing with linear probing for collision resolution

Implementation Details:

The hash table structure consists of a fixed-size header followed by a configurable number
of slots. Each slot contains a status byte (0 for empty, 1 for occupied), followed by
the fixed-size key and value.

The implementation uses linear probing for collision resolution. When the load factor
exceeds 0.7, the hash table is automatically resized to twice its original capacity
to maintain performance.
*/
package phash
