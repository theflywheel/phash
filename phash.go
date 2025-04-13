package phash

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
)

const (
	magicNumber uint32 = 0x1A2B3C4D
	version     uint32 = 1
	headerSize         = 7 * 4 // 7 uint32 fields
)

// PersistentHash is a persistent hash table implementation using memory-mapped files
type PersistentHash struct {
	mu        sync.RWMutex
	file      *os.File
	data      []byte
	filePath  string
	keySize   uint32
	valueSize uint32
	slotSize  uint32
	numSlots  uint32
	usedSlots uint32
}

// Open creates or opens a persistent hash table file
func Open(filePath string, keySize, valueSize uint32) (*PersistentHash, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	fi, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if fi.Size() == 0 {
		initialSlots := uint32(1024)
		slotSize := 1 + keySize + valueSize
		fileSize := int64(headerSize + initialSlots*slotSize)
		if err := file.Truncate(fileSize); err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to truncate file: %w", err)
		}

		header := make([]byte, headerSize)
		binary.BigEndian.PutUint32(header[0:4], magicNumber)
		binary.BigEndian.PutUint32(header[4:8], version)
		binary.BigEndian.PutUint32(header[8:12], initialSlots)
		binary.BigEndian.PutUint32(header[12:16], 0)
		binary.BigEndian.PutUint32(header[16:20], slotSize)
		binary.BigEndian.PutUint32(header[20:24], keySize)
		binary.BigEndian.PutUint32(header[24:28], valueSize)

		if _, err := file.WriteAt(header, 0); err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to write header: %w", err)
		}
	}

	// Fix for macOS: ensure file size is not zero before mmap
	fi, err = file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to re-stat file: %w", err)
	}

	fileSize := int(fi.Size())
	if fileSize == 0 {
		file.Close()
		return nil, fmt.Errorf("file size is zero after initialization")
	}

	// Use PROT_READ for compatibility
	data, err := syscall.Mmap(int(file.Fd()), 0, fileSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("mmap failed: %w", err)
	}

	magic := binary.BigEndian.Uint32(data[0:4])
	if magic != magicNumber {
		syscall.Munmap(data)
		file.Close()
		return nil, errors.New("invalid magic number")
	}

	ph := &PersistentHash{
		file:      file,
		data:      data,
		filePath:  filePath,
		keySize:   binary.BigEndian.Uint32(data[20:24]),
		valueSize: binary.BigEndian.Uint32(data[24:28]),
		slotSize:  binary.BigEndian.Uint32(data[16:20]),
		numSlots:  binary.BigEndian.Uint32(data[8:12]),
		usedSlots: binary.BigEndian.Uint32(data[12:16]),
	}

	return ph, nil
}

// Close closes the hash table and flushes changes to disk
func (ph *PersistentHash) Close() error {
	ph.mu.Lock()
	defer ph.mu.Unlock()

	if err := syscall.Munmap(ph.data); err != nil {
		return err
	}
	return ph.file.Close()
}

// Put adds or updates a key-value pair in the hash table
func (ph *PersistentHash) Put(key, value []byte) error {
	ph.mu.Lock()
	defer ph.mu.Unlock()

	if uint32(len(key)) != ph.keySize || uint32(len(value)) != ph.valueSize {
		return errors.New("invalid key/value size")
	}

	// Try to insert with retries after potential resizes
	return ph.putWithRetry(key, value, 0)
}

// putWithRetry handles the actual insertion, with a retry mechanism for resizes
func (ph *PersistentHash) putWithRetry(key, value []byte, retryCount int) error {
	// Safety check to prevent excessive recursion
	if retryCount > 3 {
		return fmt.Errorf("exceeded maximum retry count (%d) during Put operation", retryCount)
	}

	hash := hashKey(key)
	idx := hash % ph.numSlots

	for i := uint32(0); i < ph.numSlots; i++ {
		currentIdx := (idx + i) % ph.numSlots
		slotStart := headerSize + currentIdx*ph.slotSize

		switch ph.data[slotStart] {
		case 0: // Empty slot
			// Check if resize is needed
			loadFactor := float32(ph.usedSlots+1) / float32(ph.numSlots)
			if loadFactor > 0.7 {
				fmt.Printf("Resize triggered at load factor %.2f (%d/%d slots used)\n",
					loadFactor, ph.usedSlots+1, ph.numSlots)
				if err := ph.resize(); err != nil {
					return fmt.Errorf("resize failed: %w", err)
				}
				// After resize, retry the Put operation with incremented retry count
				return ph.putWithRetry(key, value, retryCount+1)
			}

			// Insert the key-value pair
			copy(ph.data[slotStart+1:], key)
			copy(ph.data[slotStart+1+ph.keySize:], value)
			ph.data[slotStart] = 1
			ph.usedSlots++
			binary.BigEndian.PutUint32(ph.data[12:16], ph.usedSlots)
			return nil

		case 1: // Occupied slot
			if bytes.Equal(key, ph.data[slotStart+1:slotStart+1+ph.keySize]) {
				// Update existing key
				copy(ph.data[slotStart+1+ph.keySize:], value)
				return nil
			}
		}
	}

	return errors.New("hash table full")
}

// Get retrieves a value from the hash table by key
func (ph *PersistentHash) Get(key []byte) ([]byte, bool) {
	ph.mu.RLock()
	defer ph.mu.RUnlock()

	if uint32(len(key)) != ph.keySize {
		return nil, false
	}

	hash := hashKey(key)
	idx := hash % ph.numSlots

	for i := uint32(0); i < ph.numSlots; i++ {
		currentIdx := (idx + i) % ph.numSlots
		slotStart := headerSize + currentIdx*ph.slotSize

		switch ph.data[slotStart] {
		case 0:
			return nil, false
		case 1:
			if bytes.Equal(key, ph.data[slotStart+1:slotStart+1+ph.keySize]) {
				val := make([]byte, ph.valueSize)
				copy(val, ph.data[slotStart+1+ph.keySize:slotStart+ph.slotSize])
				return val, true
			}
		}
	}

	return nil, false
}

func (ph *PersistentHash) resize() error {
	fmt.Printf("Starting resize: current slots=%d, used=%d\n", ph.numSlots, ph.usedSlots)

	// Use fixed increase for predictability
	newNumSlots := ph.numSlots * 2
	tmpPath := ph.filePath + ".tmp"

	// Remove any existing temporary file
	os.Remove(tmpPath)

	fmt.Printf("Creating temp file: %s with %d slots\n", tmpPath, newNumSlots)
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file for resize: %w", err)
	}
	defer tmpFile.Close()

	newSlotSize := ph.slotSize
	newFileSize := int64(headerSize + newNumSlots*newSlotSize)
	fmt.Printf("Truncating temp file to size: %d bytes\n", newFileSize)
	if err := tmpFile.Truncate(newFileSize); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to truncate temp file: %w", err)
	}

	// Write header data
	header := make([]byte, headerSize)
	binary.BigEndian.PutUint32(header[0:4], magicNumber)
	binary.BigEndian.PutUint32(header[4:8], version)
	binary.BigEndian.PutUint32(header[8:12], newNumSlots)
	binary.BigEndian.PutUint32(header[12:16], 0) // Reset used slots
	binary.BigEndian.PutUint32(header[16:20], newSlotSize)
	binary.BigEndian.PutUint32(header[20:24], ph.keySize)
	binary.BigEndian.PutUint32(header[24:28], ph.valueSize)

	fmt.Printf("Writing header to temp file\n")
	if _, err := tmpFile.WriteAt(header, 0); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write header to temp file: %w", err)
	}

	// Flush to ensure the header is written
	if err := tmpFile.Sync(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Get actual file size
	fi, err := tmpFile.Stat()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to stat temp file: %w", err)
	}
	tempFileSize := int(fi.Size())
	fmt.Printf("Actual temp file size: %d bytes\n", tempFileSize)

	// Memory map the temporary file
	fmt.Printf("Memory mapping temp file\n")
	tmpData, err := syscall.Mmap(int(tmpFile.Fd()), 0, tempFileSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to mmap temp file: %w", err)
	}
	defer syscall.Munmap(tmpData)

	fmt.Printf("Copying data to new hash table\n")
	// Rehash all existing entries
	usedCount := uint32(0)
	for i := uint32(0); i < ph.numSlots && usedCount < ph.usedSlots; i++ {
		slotStart := headerSize + i*ph.slotSize
		if ph.data[slotStart] == 1 {
			usedCount++
			key := ph.data[slotStart+1 : slotStart+1+ph.keySize]
			value := ph.data[slotStart+1+ph.keySize : slotStart+ph.slotSize]

			hash := hashKey(key)
			idx := hash % newNumSlots

			foundSlot := false
			for j := uint32(0); j < newNumSlots; j++ {
				currentIdx := (idx + j) % newNumSlots
				newSlotStart := headerSize + currentIdx*newSlotSize

				if tmpData[newSlotStart] == 0 {
					// Copy the key-value pair
					copy(tmpData[newSlotStart+1:], key)
					copy(tmpData[newSlotStart+1+ph.keySize:], value)
					tmpData[newSlotStart] = 1
					foundSlot = true

					// Update used slots count
					usedSlotsCount := binary.BigEndian.Uint32(tmpData[12:16]) + 1
					binary.BigEndian.PutUint32(tmpData[12:16], usedSlotsCount)
					break
				}
			}

			if !foundSlot {
				return fmt.Errorf("failed to find slot for key during resize")
			}
		}
	}

	// Close and unmap original file
	fmt.Printf("Unmapping and closing original file\n")
	syscall.Munmap(ph.data)
	ph.file.Close()

	// Rename temporary file to original
	fmt.Printf("Renaming temp file to original\n")
	if err := os.Rename(tmpPath, ph.filePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Reopen the file
	fmt.Printf("Reopening the file\n")
	file, err := os.OpenFile(ph.filePath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to reopen file after resize: %w", err)
	}

	// Get file size
	fi, err = file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to stat file after resize: %w", err)
	}
	fileSize := int(fi.Size())

	// Map the file
	fmt.Printf("Remapping the file, size=%d\n", fileSize)
	data, err := syscall.Mmap(int(file.Fd()), 0, fileSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to mmap file after resize: %w", err)
	}

	// Update the hash state
	ph.file = file
	ph.data = data
	ph.numSlots = newNumSlots
	ph.usedSlots = binary.BigEndian.Uint32(data[12:16])

	fmt.Printf("Resize complete: new slots=%d, used=%d\n", ph.numSlots, ph.usedSlots)
	return nil
}

const (
	offset32 = 2166136261
	prime32  = 16777619
)

// hashKey computes a 32-bit FNV-1a hash of the key
func hashKey(key []byte) uint32 {
	hash := uint32(offset32)
	for _, b := range key {
		hash ^= uint32(b)
		hash *= prime32
	}
	return hash
}
