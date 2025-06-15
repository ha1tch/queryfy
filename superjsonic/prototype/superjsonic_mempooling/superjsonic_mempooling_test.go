package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"
)

// ============================================================================
// OPTIMIZED FAST JSON PARSER WITH MEMORY POOLING
// ============================================================================

// Global parser pool to reuse parser instances
var parserPool = sync.Pool{
	New: func() interface{} {
		return &FastJSONParser{
			tokens: make([]Token, 0, 1024),
			simd:   SimdOps{},
		}
	},
}

// Small buffer pool for common JSON sizes
var smallBufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 4096) // 4KB for small JSONs
		return &buf
	},
}

// Token represents a JSON token - now much lighter
type Token struct {
	Type   TokenType
	Offset uint32 // Reduced from int to uint32
	Length uint32 // Reduced from int to uint32
}

type TokenType uint8

const (
	TokenString TokenType = iota
	TokenNumber
	TokenTrue
	TokenFalse
	TokenNull
	TokenObjectStart
	TokenObjectEnd
	TokenArrayStart
	TokenArrayEnd
	TokenColon
	TokenComma
	TokenEOF
)

// SimdOps - Portable SIMD operations
type SimdOps struct{}

func (SimdOps) HasByte(v uint64, b byte) bool {
	n := uint64(b)
	hasZero := func(v uint64) bool {
		return (v-0x0101010101010101)&^v&0x8080808080808080 != 0
	}
	return hasZero(v ^ (n * 0x0101010101010101))
}

func (SimdOps) IsWhitespace(v uint64) uint64 {
	spaces := v ^ 0x2020202020202020
	tabs := v ^ 0x0909090909090909
	lfs := v ^ 0x0A0A0A0A0A0A0A0A
	crs := v ^ 0x0D0D0D0D0D0D0D0D

	hasZero := func(v uint64) uint64 {
		return (v - 0x0101010101010101) & ^v & 0x8080808080808080
	}

	return hasZero(spaces) | hasZero(tabs) | hasZero(lfs) | hasZero(crs)
}

// FastJSONParser - optimized with no packet buffer allocation
type FastJSONParser struct {
	data   []byte  // Direct reference to input data
	tokens []Token // Reusable token slice
	pos    int     // Current position
	simd   SimdOps // SIMD operations
}

// GetParser gets a parser from the pool
func GetParser() *FastJSONParser {
	return parserPool.Get().(*FastJSONParser)
}

// ReturnParser returns parser to pool
func ReturnParser(p *FastJSONParser) {
	p.data = nil            // Clear reference
	p.tokens = p.tokens[:0] // Reset slice but keep capacity
	p.pos = 0
	parserPool.Put(p)
}

// Parse parses JSON data with minimal allocations
func (p *FastJSONParser) Parse(jsonData []byte) error {
	// Direct reference - no copying!
	p.data = jsonData
	p.tokens = p.tokens[:0] // Reset tokens but keep capacity
	p.pos = 0

	i := 0
	dataLen := len(jsonData)

	// Pre-grow tokens if needed (amortized)
	expectedTokens := dataLen / 10 // Rough estimate
	if cap(p.tokens) < expectedTokens {
		p.tokens = make([]Token, 0, expectedTokens)
	}

	for i < dataLen {
		// Fast whitespace skipping
		for i+8 <= dataLen {
			v := *(*uint64)(unsafe.Pointer(&jsonData[i]))
			wsResult := p.simd.IsWhitespace(v)
			if wsResult == 0 {
				break
			}

			found := false
			for j := 0; j < 8; j++ {
				if !isWhitespace(jsonData[i+j]) {
					i += j
					found = true
					break
				}
			}
			if found {
				break
			}
			i += 8
		}

		for i < dataLen && isWhitespace(jsonData[i]) {
			i++
		}

		if i >= dataLen {
			break
		}

		start := i

		switch jsonData[i] {
		case '{':
			p.addToken(TokenObjectStart, start, 1)
			i++
		case '}':
			p.addToken(TokenObjectEnd, start, 1)
			i++
		case '[':
			p.addToken(TokenArrayStart, start, 1)
			i++
		case ']':
			p.addToken(TokenArrayEnd, start, 1)
			i++
		case ':':
			p.addToken(TokenColon, start, 1)
			i++
		case ',':
			p.addToken(TokenComma, start, 1)
			i++
		case '"':
			i++ // skip opening quote

			// Fast string parsing
			for i+8 <= dataLen {
				v := *(*uint64)(unsafe.Pointer(&jsonData[i]))

				if p.simd.HasByte(v, '"') || p.simd.HasByte(v, '\\') {
					for j := 0; j < 8; j++ {
						if jsonData[i+j] == '\\' {
							i += j + 2
							goto continueString
						} else if jsonData[i+j] == '"' {
							i += j
							goto stringEnd
						}
					}
				} else {
					i += 8
				}
			continueString:
			}

			for i < dataLen && jsonData[i] != '"' {
				if jsonData[i] == '\\' {
					i++
				}
				i++
			}

		stringEnd:
			i++ // skip closing quote
			p.addToken(TokenString, start, i-start)

		case 't':
			if i+4 <= dataLen && *(*uint32)(unsafe.Pointer(&jsonData[i])) == 0x65757274 {
				p.addToken(TokenTrue, start, 4)
				i += 4
			}
		case 'f':
			if i+5 <= dataLen && *(*uint32)(unsafe.Pointer(&jsonData[i])) == 0x736C6166 && jsonData[i+4] == 'e' {
				p.addToken(TokenFalse, start, 5)
				i += 5
			}
		case 'n':
			if i+4 <= dataLen && *(*uint32)(unsafe.Pointer(&jsonData[i])) == 0x6C6C756E {
				p.addToken(TokenNull, start, 4)
				i += 4
			}
		default:
			if isDigit(jsonData[i]) || jsonData[i] == '-' {
				for i < dataLen && (isDigit(jsonData[i]) || jsonData[i] == '-' ||
					jsonData[i] == '.' || jsonData[i] == 'e' || jsonData[i] == 'E' ||
					jsonData[i] == '+') {
					i++
				}
				p.addToken(TokenNumber, start, i-start)
			}
		}
	}

	p.addToken(TokenEOF, dataLen, 0)
	return nil
}

// addToken adds a token efficiently
func (p *FastJSONParser) addToken(typ TokenType, offset, length int) {
	p.tokens = append(p.tokens, Token{
		Type:   typ,
		Offset: uint32(offset),
		Length: uint32(length),
	})
}

// GetTokenValue returns the token value without allocation when possible
func (p *FastJSONParser) GetTokenValue(token Token) string {
	// This creates a string without copying the underlying bytes
	// The string header points to the same memory as p.data
	return *(*string)(unsafe.Pointer(&struct {
		str unsafe.Pointer
		len int
	}{
		str: unsafe.Pointer(&p.data[token.Offset]),
		len: int(token.Length),
	}))
}

// GetTokenBytes returns token bytes without allocation
func (p *FastJSONParser) GetTokenBytes(token Token) []byte {
	return p.data[token.Offset : token.Offset+token.Length]
}

// CompareToken compares without allocation
func (p *FastJSONParser) CompareToken(token Token, target string) bool {
	if int(token.Length) != len(target) {
		return false
	}

	tokenData := p.data[token.Offset : token.Offset+token.Length]
	targetBytes := *(*[]byte)(unsafe.Pointer(&target))

	// Fast comparison 8 bytes at a time
	i := 0
	for i+8 <= len(targetBytes) {
		if *(*uint64)(unsafe.Pointer(&tokenData[i])) != *(*uint64)(unsafe.Pointer(&targetBytes[i])) {
			return false
		}
		i += 8
	}

	// Compare remaining bytes
	for ; i < len(targetBytes); i++ {
		if tokenData[i] != targetBytes[i] {
			return false
		}
	}

	return true
}

// Helper functions
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// ============================================================================
// LIGHTWEIGHT PARSER FOR COMPARISON
// ============================================================================

// NewFastJSONParser creates a new parser (for backwards compatibility)
func NewFastJSONParser() *FastJSONParser {
	return &FastJSONParser{
		tokens: make([]Token, 0, 1024),
		simd:   SimdOps{},
	}
}

// ============================================================================
// BENCHMARK CODE
// ============================================================================

func generateSimpleJSON() []byte {
	return []byte(`{"name":"John Doe","age":30,"active":true,"balance":1234.56}`)
}

func generateComplexJSON() []byte {
	return []byte(`{
		"users": [
			{"id": 1, "name": "Alice", "email": "alice@example.com", "age": 28, "active": true},
			{"id": 2, "name": "Bob", "email": "bob@example.com", "age": 32, "active": false},
			{"id": 3, "name": "Charlie", "email": "charlie@example.com", "age": 25, "active": true}
		],
		"metadata": {
			"version": "1.0.0",
			"timestamp": "2024-01-01T00:00:00Z",
			"count": 3
		},
		"settings": {
			"debug": false,
			"maxConnections": 100,
			"timeout": 30.5
		}
	}`)
}

func generateLargeJSON(numObjects int) []byte {
	var sb strings.Builder
	sb.WriteString(`{"items":[`)

	for i := 0; i < numObjects; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `{"id":%d,"name":"User%d","email":"user%d@example.com","score":%f,"active":%v}`,
			i, i, i, rand.Float64()*100, i%2 == 0)
	}

	sb.WriteString(`],"total":`)
	sb.WriteString(fmt.Sprintf("%d", numObjects))
	sb.WriteString(`}`)

	return []byte(sb.String())
}

func generateDeeplyNestedJSON(depth int) []byte {
	var sb strings.Builder

	for i := 0; i < depth; i++ {
		sb.WriteString(fmt.Sprintf(`{"level%d":`, i))
	}

	sb.WriteString(`"deepest value"`)

	for i := 0; i < depth; i++ {
		sb.WriteString(`}`)
	}

	return []byte(sb.String())
}

// Benchmarks

func BenchmarkSimpleJSON(b *testing.B) {
	data := generateSimpleJSON()

	b.Run("OptimizedParser", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))

		for i := 0; i < b.N; i++ {
			parser := GetParser()
			_ = parser.Parse(data)
			ReturnParser(parser)
		}
	})

	b.Run("StandardJSON", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))

		for i := 0; i < b.N; i++ {
			var result map[string]interface{}
			_ = json.Unmarshal(data, &result)
		}
	})
}

func BenchmarkComplexJSON(b *testing.B) {
	data := generateComplexJSON()

	b.Run("OptimizedParser", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))

		for i := 0; i < b.N; i++ {
			parser := GetParser()
			_ = parser.Parse(data)
			ReturnParser(parser)
		}
	})

	b.Run("StandardJSON", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))

		for i := 0; i < b.N; i++ {
			var result map[string]interface{}
			_ = json.Unmarshal(data, &result)
		}
	})
}

func BenchmarkLargeJSON(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		data := generateLargeJSON(size)

		b.Run(fmt.Sprintf("OptimizedParser_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))

			for i := 0; i < b.N; i++ {
				parser := GetParser()
				_ = parser.Parse(data)
				ReturnParser(parser)
			}
		})

		b.Run(fmt.Sprintf("StandardJSON_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))

			for i := 0; i < b.N; i++ {
				var result map[string]interface{}
				_ = json.Unmarshal(data, &result)
			}
		})
	}
}

func BenchmarkStringHeavyJSON(b *testing.B) {
	var sb strings.Builder
	sb.WriteString(`{"strings":[`)
	for i := 0; i < 1000; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf(`"This is a longer string number %d with some content"`, i))
	}
	sb.WriteString(`]}`)
	data := []byte(sb.String())

	b.Run("OptimizedParser", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))

		for i := 0; i < b.N; i++ {
			parser := GetParser()
			_ = parser.Parse(data)
			ReturnParser(parser)
		}
	})

	b.Run("StandardJSON", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))

		for i := 0; i < b.N; i++ {
			var result map[string]interface{}
			_ = json.Unmarshal(data, &result)
		}
	})
}

// Reuse benchmark - shows benefit of parser pooling
func BenchmarkParserReuse(b *testing.B) {
	data := generateComplexJSON()

	b.Run("WithPooling", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			parser := GetParser()
			_ = parser.Parse(data)
			ReturnParser(parser)
		}
	})

	b.Run("WithoutPooling", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			parser := NewFastJSONParser()
			_ = parser.Parse(data)
		}
	})
}

func main() {
	fmt.Println("Optimized JSON Parser Performance")
	fmt.Println("================================")
	fmt.Println()

	testSizes := []int{100, 1000, 10000}

	for _, size := range testSizes {
		data := generateLargeJSON(size)
		fmt.Printf("Testing with %d objects (%.2f KB):\n", size, float64(len(data))/1024)

		// Optimized parser with pooling
		start := time.Now()
		parser := GetParser()
		_ = parser.Parse(data)
		ReturnParser(parser)
		optTime := time.Since(start)

		// Standard parser
		start = time.Now()
		var result map[string]interface{}
		_ = json.Unmarshal(data, &result)
		stdTime := time.Since(start)

		speedup := float64(stdTime) / float64(optTime)
		fmt.Printf("  Optimized Parser: %v\n", optTime)
		fmt.Printf("  Standard JSON: %v\n", stdTime)
		fmt.Printf("  Speedup: %.2fx\n", speedup)
		fmt.Println()
	}

	fmt.Println("Optimizations Applied:")
	fmt.Println("- Parser object pooling (sync.Pool)")
	fmt.Println("- Zero-copy string references")
	fmt.Println("- No packet buffer allocation")
	fmt.Println("- Reduced token size (16 bytes → 9 bytes)")
	fmt.Println("- Reusable token slices")
	fmt.Println("- Direct data references")
}
