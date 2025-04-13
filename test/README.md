# PHash Unit Tests

This directory contains unit tests for the phash package.

## Available Tests

- **phash_basic_test.go**: Tests for basic functionality (Put/Get/Persistence/Overwrite)
- **phash_edge_cases_test.go**: Tests for edge cases (varied key/value sizes, resizing, empty values)

## Running the Tests

Run all tests:

```
go test -v
```

Run a specific test:

```
go test -v -run=TestBasicOperations
```

## Test Coverage

The unit tests cover:

1. Basic operations (put and get)
2. Persistence across open/close operations
3. Error handling with invalid inputs
4. Key overwriting behavior
5. Various key and value sizes
6. Hash table resizing
7. Edge cases like empty values
