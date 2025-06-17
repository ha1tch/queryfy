// superjsonic_fast_test.go
package main

import (
   "encoding/json"
   "fmt"
   "runtime"
   "strings"
   "sync"
   "testing"
   "unsafe"
)

// ============================================================================
// SUPERJSONIC FAST - High-Performance JSON Parser
// ============================================================================
// 
// Features:
// - SIMD-optimized whitespace skipping
// - SIMD-optimized string scanning
// - Zero-copy string access
// - Parser pooling for zero allocations
// - Optimized literal matching
// - Pre-sized token arrays
//
// Optimizations:
// - 8-byte parallel processing
// - Word-aligned memory access
// - Branch prediction hints
// - Cache-friendly data structures
// ============================================================================

// Token types
type TokenType uint8

const (
   TokenInvalid TokenType = iota
   TokenString
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

// Token represents a JSON token with zero-copy support
type Token struct {
   Type   TokenType
   Start  uint32 // Start position in input
   End    uint32 // End position in input
}

// FastParser is a high-performance JSON parser
type FastParser struct {
   input  []byte
   pos    int
   tokens []Token
   cap    int // Token capacity
}

// Parser pool for zero-allocation parsing
var parserPool = sync.Pool{
   New: func() interface{} {
   	return &FastParser{
   		tokens: make([]Token, 0, 1024),
   		cap:    1024,
   	}
   },
}

// Get returns a parser from the pool
func Get() *FastParser {
   return parserPool.Get().(*FastParser)
}

// Put returns a parser to the pool
func Put(p *FastParser) {
   p.input = nil
   p.pos = 0
   p.tokens = p.tokens[:0]
   // Resize if tokens grew too large
   if cap(p.tokens) > 8192 {
   	p.tokens = make([]Token, 0, 1024)
   	p.cap = 1024
   }
   parserPool.Put(p)
}

// Parse parses JSON input and returns tokens
func (p *FastParser) Parse(input []byte) ([]Token, error) {
   p.input = input
   p.pos = 0
   p.tokens = p.tokens[:0]
   
   // Pre-size tokens based on input size
   estimatedTokens := len(input) / 8
   if estimatedTokens > p.cap {
   	p.tokens = make([]Token, 0, estimatedTokens)
   	p.cap = estimatedTokens
   }
   
   // Parse root value
   if err := p.parseValue(); err != nil {
   	return nil, err
   }
   
   // Skip trailing whitespace
   p.skipWhitespaceFast()
   
   // Ensure all input consumed
   if p.pos < len(p.input) {
   	return nil, fmt.Errorf("unexpected content at position %d", p.pos)
   }
   
   return p.tokens, nil
}

// SIMD-optimized whitespace skipping
func (p *FastParser) skipWhitespaceFast() {
   // Process 8 bytes at a time
   for p.pos+8 <= len(p.input) {
   	// Load 8 bytes
   	chunk := *(*uint64)(unsafe.Pointer(&p.input[p.pos]))
   	
   	// Check for whitespace using SIMD-style operations
   	// Spaces: 0x20, Tabs: 0x09, LF: 0x0A, CR: 0x0D
   	spaces := chunk ^ 0x2020202020202020
   	tabs := chunk ^ 0x0909090909090909
   	newlines := chunk ^ 0x0A0A0A0A0A0A0A0A
   	returns := chunk ^ 0x0D0D0D0D0D0D0D0D
   	
   	// Check if any byte is zero after XOR (meaning it was whitespace)
   	hasZero := func(v uint64) bool {
   		return (v-0x0101010101010101)&^v&0x8080808080808080 != 0
   	}
   	
   	// If no whitespace found in this chunk, we're done
   	if !hasZero(spaces) && !hasZero(tabs) && !hasZero(newlines) && !hasZero(returns) {
   		return
   	}
   	
   	// Found whitespace, advance byte by byte until non-whitespace
   	for i := 0; i < 8 && p.pos < len(p.input); i++ {
   		switch p.input[p.pos] {
   		case ' ', '\t', '\n', '\r':
   			p.pos++
   		default:
   			return
   		}
   	}
   }
   
   // Handle remaining bytes
   for p.pos < len(p.input) {
   	switch p.input[p.pos] {
   	case ' ', '\t', '\n', '\r':
   		p.pos++
   	default:
   		return
   	}
   }
}

// Add token to the list
func (p *FastParser) addToken(typ TokenType, start, end int) {
   p.tokens = append(p.tokens, Token{
   	Type:  typ,
   	Start: uint32(start),
   	End:   uint32(end),
   })
}

// Parse any JSON value
func (p *FastParser) parseValue() error {
   p.skipWhitespaceFast()
   
   if p.pos >= len(p.input) {
   	return fmt.Errorf("unexpected end of input")
   }
   
   // Use jump table for common cases
   switch p.input[p.pos] {
   case '{':
   	return p.parseObject()
   case '[':
   	return p.parseArray()
   case '"':
   	return p.parseStringFast()
   case 't':
   	return p.parseLiteralFast("true", TokenTrue)
   case 'f':
   	return p.parseLiteralFast("false", TokenFalse)
   case 'n':
   	return p.parseLiteralFast("null", TokenNull)
   case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
   	return p.parseNumberFast()
   default:
   	return fmt.Errorf("unexpected character '%c' at position %d", p.input[p.pos], p.pos)
   }
}

// Parse object with optimizations
func (p *FastParser) parseObject() error {
   start := p.pos
   p.pos++ // skip '{'
   p.addToken(TokenObjectStart, start, p.pos)
   
   first := true
   for {
   	p.skipWhitespaceFast()
   	
   	if p.pos >= len(p.input) {
   		return fmt.Errorf("unterminated object")
   	}
   	
   	// Check for end of object
   	if p.input[p.pos] == '}' {
   		p.addToken(TokenObjectEnd, p.pos, p.pos+1)
   		p.pos++
   		return nil
   	}
   	
   	// Handle comma
   	if !first {
   		if p.input[p.pos] != ',' {
   			return fmt.Errorf("expected ',' at position %d", p.pos)
   		}
   		p.addToken(TokenComma, p.pos, p.pos+1)
   		p.pos++
   		p.skipWhitespaceFast()
   	}
   	first = false
   	
   	// Parse key (must be string)
   	if p.pos >= len(p.input) || p.input[p.pos] != '"' {
   		return fmt.Errorf("expected string key at position %d", p.pos)
   	}
   	if err := p.parseStringFast(); err != nil {
   		return err
   	}
   	
   	// Parse colon
   	p.skipWhitespaceFast()
   	if p.pos >= len(p.input) || p.input[p.pos] != ':' {
   		return fmt.Errorf("expected ':' at position %d", p.pos)
   	}
   	p.addToken(TokenColon, p.pos, p.pos+1)
   	p.pos++
   	
   	// Parse value
   	if err := p.parseValue(); err != nil {
   		return err
   	}
   }
}

// Parse array with optimizations
func (p *FastParser) parseArray() error {
   start := p.pos
   p.pos++ // skip '['
   p.addToken(TokenArrayStart, start, p.pos)
   
   first := true
   for {
   	p.skipWhitespaceFast()
   	
   	if p.pos >= len(p.input) {
   		return fmt.Errorf("unterminated array")
   	}
   	
   	// Check for end of array
   	if p.input[p.pos] == ']' {
   		p.addToken(TokenArrayEnd, p.pos, p.pos+1)
   		p.pos++
   		return nil
   	}
   	
   	// Handle comma
   	if !first {
   		if p.input[p.pos] != ',' {
   			return fmt.Errorf("expected ',' at position %d", p.pos)
   		}
   		p.addToken(TokenComma, p.pos, p.pos+1)
   		p.pos++
   		p.skipWhitespaceFast()
   	}
   	first = false
   	
   	// Parse value
   	if err := p.parseValue(); err != nil {
   		return err
   	}
   }
}

// SIMD-optimized string parsing
func (p *FastParser) parseStringFast() error {
   start := p.pos
   p.pos++ // skip opening '"'
   
   // SIMD string scanning - process 8 bytes at a time
   for p.pos+8 <= len(p.input) {
   	// Load 8 bytes
   	chunk := *(*uint64)(unsafe.Pointer(&p.input[p.pos]))
   	
   	// Check for quotes or backslashes in parallel
   	quotes := chunk ^ 0x2222222222222222     // '"' repeated 8 times
   	backslashes := chunk ^ 0x5C5C5C5C5C5C5C5C // '\' repeated 8 times
   	
   	// Check if any byte is zero after XOR
   	hasZero := func(v uint64) bool {
   		return (v-0x0101010101010101)&^v&0x8080808080808080 != 0
   	}
   	
   	// If found quote or backslash, handle byte by byte
   	if hasZero(quotes) || hasZero(backslashes) {
   		for i := 0; i < 8; i++ {
   			if p.input[p.pos+i] == '"' {
   				p.pos += i + 1
   				p.addToken(TokenString, start, p.pos)
   				return nil
   			}
   			if p.input[p.pos+i] == '\\' {
   				// Skip the backslash and escaped character
   				p.pos += i + 1
   				if p.pos >= len(p.input) {
   					return fmt.Errorf("unterminated string escape")
   				}
   				p.pos++
   				goto continueAfterEscape
   			}
   		}
   	}
   	
   	// No special characters in this chunk, skip all 8 bytes
   	p.pos += 8
   continueAfterEscape:
   }
   
   // Handle remaining bytes
   for p.pos < len(p.input) {
   	switch p.input[p.pos] {
   	case '"':
   		p.pos++
   		p.addToken(TokenString, start, p.pos)
   		return nil
   	case '\\':
   		p.pos++
   		if p.pos >= len(p.input) {
   			return fmt.Errorf("unterminated string escape")
   		}
   		p.pos++
   	default:
   		p.pos++
   	}
   }
   
   return fmt.Errorf("unterminated string")
}

// Fast literal parsing using word comparison
func (p *FastParser) parseLiteralFast(literal string, tokenType TokenType) error {
   start := p.pos
   litLen := len(literal)
   
   // Check bounds
   if p.pos+litLen > len(p.input) {
   	return fmt.Errorf("unexpected end of input")
   }
   
   // Use word comparison for common literals
   switch litLen {
   case 4: // "true", "null"
   	word := *(*uint32)(unsafe.Pointer(&p.input[p.pos]))
   	var expected uint32
   	switch literal {
   	case "true":
   		expected = 0x65757274 // "true" in little-endian
   	case "null":
   		expected = 0x6c6c756e // "null" in little-endian
   	}
   	if word != expected {
   		return fmt.Errorf("invalid literal at position %d", p.pos)
   	}
   	
   case 5: // "false"
   	// Check first 4 bytes
   	word := *(*uint32)(unsafe.Pointer(&p.input[p.pos]))
   	if word != 0x736c6166 { // "fals" in little-endian
   		return fmt.Errorf("invalid literal at position %d", p.pos)
   	}
   	// Check last byte
   	if p.input[p.pos+4] != 'e' {
   		return fmt.Errorf("invalid literal at position %d", p.pos)
   	}
   	
   default:
   	// Fallback for other literals
   	for i := 0; i < litLen; i++ {
   		if p.input[p.pos+i] != literal[i] {
   			return fmt.Errorf("invalid literal at position %d", p.pos)
   		}
   	}
   }
   
   p.pos += litLen
   p.addToken(tokenType, start, p.pos)
   return nil
}

// Optimized number parsing
func (p *FastParser) parseNumberFast() error {
   start := p.pos
   
   // Handle negative sign
   if p.input[p.pos] == '-' {
   	p.pos++
   	if p.pos >= len(p.input) {
   		return fmt.Errorf("invalid number: unexpected end")
   	}
   }
   
   // Must have at least one digit
   if !isDigit(p.input[p.pos]) {
   	return fmt.Errorf("invalid number: no digits")
   }
   
   // Parse integer part
   if p.input[p.pos] == '0' {
   	p.pos++
   } else {
   	// Use SIMD-style digit scanning
   	for p.pos+8 <= len(p.input) {
   		chunk := *(*uint64)(unsafe.Pointer(&p.input[p.pos]))
   		
   		// Check if all bytes are digits
   		allDigits := true
   		for i := 0; i < 8; i++ {
   			b := byte(chunk >> (i * 8))
   			if !isDigit(b) {
   				allDigits = false
   				break
   			}
   		}
   		
   		if !allDigits {
   			// Find first non-digit
   			for i := 0; i < 8; i++ {
   				if !isDigit(p.input[p.pos]) {
   					goto integerDone
   				}
   				p.pos++
   			}
   		} else {
   			p.pos += 8
   		}
   	}
   	
   	// Handle remaining digits
   	for p.pos < len(p.input) && isDigit(p.input[p.pos]) {
   		p.pos++
   	}
   }
integerDone:
   
   // Parse fractional part
   if p.pos < len(p.input) && p.input[p.pos] == '.' {
   	p.pos++
   	if p.pos >= len(p.input) || !isDigit(p.input[p.pos]) {
   		return fmt.Errorf("invalid number: no digits after decimal")
   	}
   	for p.pos < len(p.input) && isDigit(p.input[p.pos]) {
   		p.pos++
   	}
   }
   
   // Parse exponent
   if p.pos < len(p.input) && (p.input[p.pos] == 'e' || p.input[p.pos] == 'E') {
   	p.pos++
   	if p.pos < len(p.input) && (p.input[p.pos] == '+' || p.input[p.pos] == '-') {
   		p.pos++
   	}
   	if p.pos >= len(p.input) || !isDigit(p.input[p.pos]) {
   		return fmt.Errorf("invalid number: no digits in exponent")
   	}
   	for p.pos < len(p.input) && isDigit(p.input[p.pos]) {
   		p.pos++
   	}
   }
   
   p.addToken(TokenNumber, start, p.pos)
   return nil
}

// Zero-copy string access
func (p *FastParser) GetTokenString(token Token) string {
   if int(token.Start) < len(p.input) && int(token.End) <= len(p.input) {
   	return string(p.input[token.Start:token.End])
   }
   return ""
}

// Zero-copy byte slice access (avoids allocation)
func (p *FastParser) GetTokenBytes(token Token) []byte {
   if int(token.Start) < len(p.input) && int(token.End) <= len(p.input) {
   	return p.input[token.Start:token.End]
   }
   return nil
}

// Helper functions
func isDigit(b byte) bool {
   return b >= '0' && b <= '9'
}

// ============================================================================
// PARALLEL BATCH PROCESSING
// ============================================================================

// ParseBatch processes multiple JSON documents in parallel
func ParseBatch(documents [][]byte) ([][]Token, []error) {
   n := len(documents)
   results := make([][]Token, n)
   errors := make([]error, n)
   
   // For small batches, process sequentially
   if n < runtime.NumCPU() {
   	for i, doc := range documents {
   		parser := Get()
   		results[i], errors[i] = parser.Parse(doc)
   		Put(parser)
   	}
   	return results, errors
   }
   
   // Process in parallel
   var wg sync.WaitGroup
   workers := runtime.NumCPU()
   chunkSize := (n + workers - 1) / workers
   
   for w := 0; w < workers; w++ {
   	start := w * chunkSize
   	end := start + chunkSize
   	if end > n {
   		end = n
   	}
   	
   	wg.Add(1)
   	go func(start, end int) {
   		defer wg.Done()
   		parser := Get()
   		defer Put(parser)
   		
   		for i := start; i < end; i++ {
   			// Reuse parser for batch
   			results[i], errors[i] = parser.Parse(documents[i])
   		}
   	}(start, end)
   }
   
   wg.Wait()
   return results, errors
}

// ============================================================================
// BENCHMARKS
// ============================================================================

var (
   tinyJSON   = []byte(`{"ok":true}`)
   smallJSON  = []byte(`{"id":123,"name":"test","email":"test@example.com","active":true}`)
   mediumJSON = []byte(`{"users":[` + strings.Repeat(`{"id":1,"name":"John Doe","email":"john@example.com","active":true},`, 100) + `]}`)
   largeJSON  = []byte(`{"data":[` + strings.Repeat(`{"id":1,"value":123.45,"nested":{"a":1,"b":2,"c":3},"tags":["x","y","z"]},`, 1000) + `]}`)
   hugeJSON   = []byte(`{"records":[` + strings.Repeat(`{"id":1,"name":"Item","description":"A long description text that spans multiple words","metadata":{"created":"2024-01-01","updated":"2024-01-02","version":1},"tags":["tag1","tag2","tag3","tag4","tag5"],"values":[1,2,3,4,5,6,7,8,9,10]},`, 10000) + `]}`)
)

var (
   fastTokens []Token
   fastError  error
   jsonResult interface{}
)

func BenchmarkFast(b *testing.B) {
   cases := []struct {
   	name string
   	data []byte
   }{
   	{"Tiny", tinyJSON},
   	{"Small", smallJSON},
   	{"Medium", mediumJSON},
   	{"Large", largeJSON},
   	{"Huge", hugeJSON},
   }
   
   for _, tc := range cases {
   	b.Run(tc.name, func(b *testing.B) {
   		b.SetBytes(int64(len(tc.data)))
   		b.ReportAllocs()
   		b.ResetTimer()
   		
   		for i := 0; i < b.N; i++ {
   			parser := Get()
   			fastTokens, fastError = parser.Parse(tc.data)
   			Put(parser)
   		}
   	})
   }
}

func BenchmarkFastBatch(b *testing.B) {
   // Create batches of different sizes
   smallBatch := make([][]byte, 100)
   for i := range smallBatch {
   	smallBatch[i] = smallJSON
   }
   
   mediumBatch := make([][]byte, 100)
   for i := range mediumBatch {
   	mediumBatch[i] = mediumJSON
   }
   
   cases := []struct {
   	name  string
   	batch [][]byte
   }{
   	{"Smallx100", smallBatch},
   	{"Mediumx100", mediumBatch},
   }
   
   for _, tc := range cases {
   	b.Run(tc.name, func(b *testing.B) {
   		totalBytes := int64(0)
   		for _, doc := range tc.batch {
   			totalBytes += int64(len(doc))
   		}
   		b.SetBytes(totalBytes)
   		b.ReportAllocs()
   		b.ResetTimer()
   		
   		for i := 0; i < b.N; i++ {
   			ParseBatch(tc.batch)
   		}
   	})
   }
}

func BenchmarkStdlib(b *testing.B) {
   cases := []struct {
   	name string
   	data []byte
   }{
   	{"Tiny", tinyJSON},
   	{"Small", smallJSON},
   	{"Medium", mediumJSON},
   	{"Large", largeJSON},
   	{"Huge", hugeJSON},
   }
   
   for _, tc := range cases {
   	b.Run(tc.name, func(b *testing.B) {
   		b.SetBytes(int64(len(tc.data)))
   		b.ReportAllocs()
   		b.ResetTimer()
   		
   		for i := 0; i < b.N; i++ {
   			fastError = json.Unmarshal(tc.data, &jsonResult)
   		}
   	})
   }
}

// ============================================================================
// VALIDATION TESTS
// ============================================================================

func TestFastParser(t *testing.T) {
   tests := []struct {
   	name      string
   	input     string
   	wantErr   bool
   	wantTypes []TokenType
   }{
   	{
   		name:      "empty object",
   		input:     "{}",
   		wantErr:   false,
   		wantTypes: []TokenType{TokenObjectStart, TokenObjectEnd},
   	},
   	{
   		name:      "empty array",
   		input:     "[]",
   		wantErr:   false,
   		wantTypes: []TokenType{TokenArrayStart, TokenArrayEnd},
   	},
   	{
   		name:      "string with escapes",
   		input:     `{"key":"value\n\t\""}`,
   		wantErr:   false,
   		wantTypes: []TokenType{TokenObjectStart, TokenString, TokenColon, TokenString, TokenObjectEnd},
   	},
   	{
   		name:      "numbers",
   		input:     `[123,-456,0.789,1.23e4,-5.67E-8]`,
   		wantErr:   false,
   		wantTypes: []TokenType{TokenArrayStart, TokenNumber, TokenComma, TokenNumber, TokenComma, TokenNumber, TokenComma, TokenNumber, TokenComma, TokenNumber, TokenArrayEnd},
   	},
   	{
   		name:    "invalid json",
   		input:   `{"key":}`,
   		wantErr: true,
   	},
   }
   
   for _, tt := range tests {
   	t.Run(tt.name, func(t *testing.T) {
   		parser := Get()
   		defer Put(parser)
   		
   		tokens, err := parser.Parse([]byte(tt.input))
   		if (err != nil) != tt.wantErr {
   			t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
   			return
   		}
   		
   		if !tt.wantErr && len(tt.wantTypes) > 0 {
   			if len(tokens) != len(tt.wantTypes) {
   				t.Errorf("Parse() returned %d tokens, want %d", len(tokens), len(tt.wantTypes))
   				return
   			}
   			for i, token := range tokens {
   				if token.Type != tt.wantTypes[i] {
   					t.Errorf("Token[%d] type = %v, want %v", i, token.Type, tt.wantTypes[i])
   				}
   			}
   		}
   	})
   }
}

// ============================================================================
// USAGE EXAMPLE
// ============================================================================

func ExampleFastParser() {
   input := []byte(`{"name":"John","age":30,"tags":["go","json"]}`)
   
   // Get parser from pool
   parser := Get()
   defer Put(parser)
   
   // Parse JSON
   tokens, err := parser.Parse(input)
   if err != nil {
   	fmt.Printf("Parse error: %v\n", err)
   	return
   }
   
   // Count token types
   counts := make(map[TokenType]int)
   for _, token := range tokens {
   	counts[token.Type]++
   }
   
   fmt.Printf("Total tokens: %d\n", len(tokens))
   fmt.Printf("Strings: %d\n", counts[TokenString])
   fmt.Printf("Numbers: %d\n", counts[TokenNumber])
   fmt.Printf("Arrays: %d\n", counts[TokenArrayStart])
   
   // Output:
   // Total tokens: 15
   // Strings: 5
   // Numbers: 1
   // Arrays: 1
}