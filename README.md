# PersistentHash

A high-performance persistent hash table for fixed key/value sizes. [Benchmark results](https://github.com/theflywheel/phash/blob/main/benchmark_history/latest.json)

## Contributing

Raising issues and PRs are welcome. Proof points of performance are expected with PRs.

## Installation

```
go get github.com/theflywheel/phash
```

## Usage

```go
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
    fmt.Println("Value:", val) // Output: Value: 67890
}
```

### Run benchmarks

```bash
./bench/tools/run_benchmarks.sh
```

## Next Steps

- [ ] Write a blog post about the implementation
- [ ] Add a small redis style I/O layer on top of the hash. SET, GET, DEL, KEYS, etc.
- [ ] Add a benchmark from Redis
- [ ] Moar tests

## License

MIT
