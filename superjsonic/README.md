# Superjsonic

A high-performance JSON parser for Go that provides fast validation through zero-allocation tokenization.

## Status
# WORK IN PROGRESS

## Overview

Superjsonic is a specialized JSON parser designed for validation scenarios where you need to verify JSON structure and content without the overhead of creating objects. It achieves 5-11x performance improvements over `encoding/json` by using a token-based approach with zero memory allocations during parsing.

## Key Features

- **Zero allocations** during parsing - no garbage collection pressure
- **Token-based parsing** - validate structure without building object trees
- **SIMD-optimized** - fast whitespace and string scanning
- **Parser pooling** - efficient reuse across requests
- **Direct byte access** - no string copies or unnecessary conversions

## Performance

Benchmarks on Apple M1 (arm64):

| Scenario | Size | encoding/json | Superjsonic | Speedup | Allocations |
|----------|------|---------------|-------------|---------|-------------|
| Simple Object | 60B | 1,190 ns/op | 110 ns/op | **10.9x** | 0 |
| Complex Object | 468B | 4,903 ns/op | 828 ns/op | **5.9x** | 0 |
| Array (1K items) | 88KB | 2,247 µs/op | 551 µs/op | **4.1x** | 0 |
| Array (100K items) | 9.4MB | 84.7 ms/op | 16.2 ms/op | **5.2x** | 0 |
| String-heavy | 55KB | 265 ms/op | 19 ms/op | **14.2x** | 0 |


## Documentation

For detailed documentation about Superjsonic, see:

- **[Understanding Superjsonic](https://github.com/ha1tch/queryfy/blob/main/superjsonic/doc/superjsonic-01-understanding-EN.md)** - Comprehensive guide to how Superjsonic works and why it's fast
- **[Architecture](https://github.com/ha1tch/queryfy/blob/main/superjsonic/doc/superjsonic-02-architecture.md)** - Technical architecture and design decisions
- **[API Reference](https://github.com/ha1tch/queryfy/blob/main/superjsonic/doc/superjsonic-03-api.md)** - Complete API documentation
- **[Integration Guide](https://github.com/ha1tch/queryfy/blob/main/superjsonic/doc/superjsonic-05-integration-with-queryfy.md)** - How Superjsonic integrates with Queryfy
- **[Real-World Use Cases](https://github.com/ha1tch/queryfy/blob/main/superjsonic/doc/superjsonic-06-real-world-problems.md)** - Problems Superjsonic solves in production

También disponible en español: **[Entendiendo Superjsonic](https://github.com/ha1tch/queryfy/blob/main/superjsonic/doc/superjsonic-01-understanding-ES.md)**

## Installation

Superjsonic is included as part of the Queryfy package:

```bash
go get github.com/ha1tch/queryfy/superjsonic
```

## Usage

### Basic Parsing
This is the expected syntax that will be used once full integration with Queryfy is completed

```go
import "github.com/ha1tch/queryfy/superjsonic"

// Get a parser from the pool
parser := superjsonic.GetParser()
defer superjsonic.ReturnParser(parser)

// Parse JSON data
jsonData := []byte(`{"name": "John", "age": 30}`)
err := parser.Parse(jsonData)
if err != nil {
    log.Fatal(err)
}

// Access tokens
for _, token := range parser.Tokens() {
    value := parser.GetTokenValue(token)
    fmt.Printf("Token: %v, Value: %s\n", token.Type, value)
}
```

### Integration with Validation

Superjsonic is designed to work with validation libraries like Queryfy:

```go
// Parse once
parser := superjsonic.GetParser()
err := parser.Parse(jsonData)
if err != nil {
    return fmt.Errorf("invalid JSON: %w", err)
}

// Validate tokens against schema
err = validator.ValidateTokens(parser.Tokens(), schema)
superjsonic.ReturnParser(parser)

if err != nil {
    return fmt.Errorf("validation failed: %w", err)
}
```

### Token Types

Superjsonic identifies these JSON elements:

- `TokenObjectStart` / `TokenObjectEnd` - Object boundaries `{}`
- `TokenArrayStart` / `TokenArrayEnd` - Array boundaries `[]`
- `TokenString` - String values (including field names)
- `TokenNumber` - Numeric values
- `TokenTrue` / `TokenFalse` / `TokenNull` - Literals
- `TokenColon` / `TokenComma` - Structural elements

## Design Philosophy

Superjsonic follows a focused design philosophy:

1. **Do one thing well** - Parse JSON into tokens for validation
2. **Zero allocations** - Never allocate memory during parsing
3. **Direct access** - Work with original bytes, not copies
4. **Pool everything** - Reuse parser instances

This is not a general-purpose JSON library. It's specifically optimized for validation scenarios where you need to check JSON structure and content without unmarshaling into objects.

## When to Use Superjsonic

**Use Superjsonic when:**
- Validating JSON before processing
- Checking JSON structure without needing the data
- Building high-performance API gateways
- Processing large volumes of JSON

**Use encoding/json when:**
- You need to unmarshal JSON into structs
- Performance is not critical
- You need full JSON specification compliance
- Working with streaming JSON

## Benchmarks

You may run the benchmkarks of the existing prototype code in the subdirectories ./superjsonic/prototype, each with:

```bash
./runbench.sh
```

The prototype benchmarks include:
- Simple and complex object parsing
- Large array handling (up to 100K elements)
- String-heavy JSON
- Concurrent parsing scenarios

## Implementation Details

### Zero Allocation Strategy

Superjsonic achieves zero allocations through:
- Pre-allocated token arrays
- Direct byte slice references (no string copies)
- Parser object pooling
- Careful memory layout (Token struct is 12 bytes)

### SIMD Optimizations

When scanning for delimiters and whitespace, Superjsonic processes 8 bytes at a time using portable SIMD operations that work across architectures.

### Array Detection

For large arrays, Superjsonic automatically detects patterns like `{"items":[...]}` and optimizes token allocation accordingly.

## License

Apache 2.0 - see LICENSE file for details
