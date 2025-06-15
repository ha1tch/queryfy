package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"
)

// ============================================================================
// OPTIMIZED SUPERJSONIC PARSER WITH ARRAY DETECTION
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

// ArrayInfo holds information about detected array patterns
type ArrayInfo struct {
	isLargeArray      bool
	tokensPerElement  int
	estimatedElements int
	firstElementEnd   int
}

// FastJSONParser - optimized with array detection
type FastJSONParser struct {
	data      []byte    // Direct reference to input data
	tokens    []Token   // Reusable token slice
	pos       int       // Current position
	simd      SimdOps   // SIMD operations
	arrayInfo ArrayInfo // Array optimization info
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
	p.arrayInfo = ArrayInfo{} // Reset array info
	parserPool.Put(p)
}

// detectLargeArray checks if JSON likely contains a large array
func (p *FastJSONParser) detectLargeArray(data []byte) ArrayInfo {
	info := ArrayInfo{}

	// Skip initial whitespace
	i := 0
	for i < len(data) && isWhitespace(data[i]) {
		i++
	}

	// Common patterns for large arrays:
	// 1. {"items":[ or {"data":[ or {"results":[
	// 2. Just starts with [

	if i < len(data) && data[i] == '[' {
		// Direct array
		info.isLargeArray = true
		info.estimatedElements = p.estimateArraySize(data, i)
		return info
	}

	// Check for object with array field
	if i < len(data) && data[i] == '{' {
		// Look for common array field names
		arrayFieldNames := [][]byte{
			[]byte(`"items":[`),
			[]byte(`"data":[`),
			[]byte(`"results":[`),
			[]byte(`"records":[`),
			[]byte(`"rows":[`),
			[]byte(`"users":[`),
			[]byte(`"products":[`),
		}

		// Quick scan for array patterns (check first 200 bytes)
		scanEnd := i + 200
		if scanEnd > len(data) {
			scanEnd = len(data)
		}

		for _, pattern := range arrayFieldNames {
			if idx := bytesIndexInRange(data, pattern, i, scanEnd); idx >= 0 {
				info.isLargeArray = true
				arrayStart := idx + len(pattern) - 1
				info.estimatedElements = p.estimateArraySize(data, arrayStart)

				// Parse first element to count tokens
				if arrayStart < len(data)-1 && data[arrayStart] == '[' {
					info.firstElementEnd = p.findFirstElementEnd(data, arrayStart+1)
					if info.firstElementEnd > 0 {
						// Count tokens in first element
						tempParser := &FastJSONParser{data: data, simd: SimdOps{}}
						tempParser.parseRange(arrayStart+1, info.firstElementEnd)
						info.tokensPerElement = len(tempParser.tokens)
					}
				}
				return info
			}
		}
	}

	return info
}

// estimateArraySize estimates the number of elements based on file size
func (p *FastJSONParser) estimateArraySize(data []byte, arrayStart int) int {
	// Sample first few elements to estimate average size
	commaCount := 0
	braceDepth := 0
	inString := false

	// Scan first 10KB to estimate
	scanEnd := arrayStart + 10240
	if scanEnd > len(data) {
		scanEnd = len(data)
	}

	for i := arrayStart; i < scanEnd; i++ {
		if !inString {
			switch data[i] {
			case '"':
				inString = true
			case '{', '[':
				braceDepth++
			case '}', ']':
				braceDepth--
			case ',':
				if braceDepth == 1 { // Only count top-level commas
					commaCount++
				}
			}
		} else if data[i] == '"' && (i == arrayStart || data[i-1] != '\\') {
			inString = false
		}
	}

	if commaCount > 0 {
		avgElementSize := (scanEnd - arrayStart) / (commaCount + 1)
		estimatedElements := len(data) / avgElementSize
		return estimatedElements
	}

	// Default estimate
	return len(data) / 50 // Assume 50 bytes per element average
}

// findFirstElementEnd finds the end of the first array element
func (p *FastJSONParser) findFirstElementEnd(data []byte, start int) int {
	braceDepth := 0
	inString := false

	for i := start; i < len(data); i++ {
		if !inString {
			switch data[i] {
			case '"':
				inString = true
			case '{', '[':
				braceDepth++
			case '}', ']':
				braceDepth--
				if braceDepth == 0 && (i+1 < len(data) && (data[i+1] == ',' || isWhitespace(data[i+1]))) {
					return i + 1
				}
			case ',':
				if braceDepth == 0 {
					return i
				}
			}
		} else if data[i] == '"' && (i == start || data[i-1] != '\\') {
			inString = false
		}
	}

	return -1
}

// parseRange parses a specific range of the JSON
func (p *FastJSONParser) parseRange(start, end int) {
	i := start
	dataLen := end

	for i < dataLen {
		// Skip whitespace using regular method for small ranges
		for i < dataLen && isWhitespace(p.data[i]) {
			i++
		}

		if i >= dataLen {
			break
		}

		tokenStart := i

		switch p.data[i] {
		case '{':
			p.addToken(TokenObjectStart, tokenStart, 1)
			i++
		case '}':
			p.addToken(TokenObjectEnd, tokenStart, 1)
			i++
		case '[':
			p.addToken(TokenArrayStart, tokenStart, 1)
			i++
		case ']':
			p.addToken(TokenArrayEnd, tokenStart, 1)
			i++
		case ':':
			p.addToken(TokenColon, tokenStart, 1)
			i++
		case ',':
			p.addToken(TokenComma, tokenStart, 1)
			i++
		case '"':
			i++ // skip opening quote
			for i < dataLen && p.data[i] != '"' {
				if p.data[i] == '\\' {
					i++
				}
				i++
			}
			i++ // skip closing quote
			p.addToken(TokenString, tokenStart, i-tokenStart)
		case 't':
			if i+4 <= dataLen && *(*uint32)(unsafe.Pointer(&p.data[i])) == 0x65757274 {
				p.addToken(TokenTrue, tokenStart, 4)
				i += 4
			}
		case 'f':
			if i+5 <= dataLen && *(*uint32)(unsafe.Pointer(&p.data[i])) == 0x736C6166 && p.data[i+4] == 'e' {
				p.addToken(TokenFalse, tokenStart, 5)
				i += 5
			}
		case 'n':
			if i+4 <= dataLen && *(*uint32)(unsafe.Pointer(&p.data[i])) == 0x6C6C756E {
				p.addToken(TokenNull, tokenStart, 4)
				i += 4
			}
		default:
			if isDigit(p.data[i]) || p.data[i] == '-' {
				for i < dataLen && (isDigit(p.data[i]) || p.data[i] == '-' ||
					p.data[i] == '.' || p.data[i] == 'e' || p.data[i] == 'E' ||
					p.data[i] == '+') {
					i++
				}
				p.addToken(TokenNumber, tokenStart, i-tokenStart)
			}
		}
	}
}

// Parse parses JSON data with array optimization
func (p *FastJSONParser) Parse(jsonData []byte) error {
	// Direct reference - no copying!
	p.data = jsonData
	p.tokens = p.tokens[:0] // Reset tokens but keep capacity
	p.pos = 0

	// Detect if this is a large array
	p.arrayInfo = p.detectLargeArray(jsonData)

	if p.arrayInfo.isLargeArray && p.arrayInfo.estimatedElements > 1000 {
		// Use optimized array parsing
		return p.parseOptimizedArray(jsonData)
	}

	// Use standard parsing for non-array or small JSONs
	return p.parseStandard(jsonData)
}

// parseOptimizedArray uses specialized parsing for large arrays
func (p *FastJSONParser) parseOptimizedArray(jsonData []byte) error {
	dataLen := len(jsonData)

	// Pre-allocate tokens based on estimation
	estimatedTokens := p.arrayInfo.estimatedElements * p.arrayInfo.tokensPerElement
	if estimatedTokens == 0 {
		// Fallback estimation: assume 5 tokens per element
		estimatedTokens = p.arrayInfo.estimatedElements * 5
	}

	// Add some buffer for container tokens
	estimatedTokens += 100

	if cap(p.tokens) < estimatedTokens {
		p.tokens = make([]Token, 0, estimatedTokens)
	}

	// Use standard parsing but with better pre-allocation
	i := 0

	for i < dataLen {
		// Fast whitespace skipping using SIMD
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
			// Fast string parsing with SIMD
			i++ // skip opening quote

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

// parseStandard is the original parsing method
func (p *FastJSONParser) parseStandard(jsonData []byte) error {
	i := 0
	dataLen := len(jsonData)

	// Standard pre-growth for smaller files
	expectedTokens := dataLen / 10
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
	return *(*string)(unsafe.Pointer(&struct {
		str unsafe.Pointer
		len int
	}{
		str: unsafe.Pointer(&p.data[token.Offset]),
		len: int(token.Length),
	}))
}

// Helper functions
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// bytesIndexInRange searches for pattern in data within a range
func bytesIndexInRange(data, pattern []byte, start, end int) int {
	if len(pattern) == 0 || start >= end {
		return -1
	}

	for i := start; i <= end-len(pattern); i++ {
		found := true
		for j := 0; j < len(pattern); j++ {
			if data[i+j] != pattern[j] {
				found = false
				break
			}
		}
		if found {
			return i
		}
	}
	return -1
}

// NewFastJSONParser creates a new parser (for backwards compatibility)
func NewFastJSONParser() *FastJSONParser {
	return &FastJSONParser{
		tokens: make([]Token, 0, 1024),
		simd:   SimdOps{},
	}
}

// ============================================================================
// TEST GENERATORS
// ============================================================================

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

// ============================================================================
// BENCHMARKS
// ============================================================================

func BenchmarkOptimizedLargeArrays(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		data := generateLargeJSON(size)

		b.Run(fmt.Sprintf("Optimized_%d", size), func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				parser := GetParser()
				_ = parser.Parse(data)
				ReturnParser(parser)
			}
		})

		b.Run(fmt.Sprintf("StandardJSON_%d", size), func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var result map[string]interface{}
				_ = json.Unmarshal(data, &result)
			}
		})
	}
}

// ============================================================================
// MAIN FUNCTION FOR TESTING
// ============================================================================

func main() {
	fmt.Println("SUPERJSONIC ARRAY-OPTIMIZED PARSER TEST")
	fmt.Println("=======================================")

	// Test array detection
	testData := []struct {
		name string
		json []byte
	}{
		{"Direct Array", []byte(`[{"id":1},{"id":2}]`)},
		{"Items Array", generateLargeJSON(10)},
		{"Data Array", []byte(`{"data":[{"x":1},{"x":2}]}`)},
		{"Non-Array", []byte(`{"name":"test","value":123}`)},
	}

	for _, test := range testData {
		parser := GetParser()
		info := parser.detectLargeArray(test.json)
		fmt.Printf("\n%s:\n", test.name)
		fmt.Printf("  Is Large Array: %v\n", info.isLargeArray)
		fmt.Printf("  Estimated Elements: %d\n", info.estimatedElements)
		fmt.Printf("  Tokens Per Element: %d\n", info.tokensPerElement)
		ReturnParser(parser)
	}

	// Performance test
	fmt.Println("\n=== Performance Comparison ===")

	testSizes := []int{100, 1000, 10000, 100000}

	for _, size := range testSizes {
		data := generateLargeJSON(size)
		fmt.Printf("\nArray with %d objects (%.2f KB):\n", size, float64(len(data))/1024)

		// Optimized parser
		start := time.Now()
		parser := GetParser()
		_ = parser.Parse(data)
		optimizedTime := time.Since(start)
		tokenCount := len(parser.tokens)
		wasOptimized := parser.arrayInfo.isLargeArray && size > 1000
		ReturnParser(parser)

		// Standard JSON
		start = time.Now()
		var result map[string]interface{}
		_ = json.Unmarshal(data, &result)
		stdTime := time.Since(start)

		speedup := float64(stdTime) / float64(optimizedTime)
		fmt.Printf("  Optimized: %v (%d tokens, %.2f MB/s) [Array-optimized: %v]\n",
			optimizedTime, tokenCount,
			float64(len(data))/optimizedTime.Seconds()/1024/1024,
			wasOptimized)
		fmt.Printf("  Standard: %v (%.2f MB/s)\n",
			stdTime,
			float64(len(data))/stdTime.Seconds()/1024/1024)
		fmt.Printf("  Speedup: %.2fx\n", speedup)
	}

	// Memory stats
	fmt.Println("\n=== Memory Usage ===")
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	fmt.Printf("Alloc = %v MB, TotalAlloc = %v MB\n",
		m.Alloc/1024/1024, m.TotalAlloc/1024/1024)
}
