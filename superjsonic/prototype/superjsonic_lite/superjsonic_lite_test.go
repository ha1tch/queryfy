// superjsonic_lite_test.go
package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// ============================================================================
// SUPERJSONIC LITE - Minimal JSON Parser
// ============================================================================
//
// Features:
// - Pure Go implementation (no unsafe)
// - Minimal memory footprint
// - Single-pass tokenization
// - No external dependencies
// - Suitable for embedded systems
// - Easy to understand and modify
//
// Trade-offs:
// - No SIMD optimizations
// - No parser pooling
// - No zero-copy strings
// - Simpler but slower than Fast version
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

// Token represents a JSON token
type Token struct {
	Type   TokenType
	Start  int // Start position in input
	End    int // End position in input
	Parent int // Index of parent token (-1 for root)
}

// LiteParser is a minimal JSON parser
type LiteParser struct {
	input  []byte
	pos    int
	tokens []Token
	stack  []int // Stack of token indices for nesting
}

// NewLiteParser creates a new parser instance
func NewLiteParser() *LiteParser {
	return &LiteParser{
		tokens: make([]Token, 0, 64),
		stack:  make([]int, 0, 16),
	}
}

// Parse parses JSON input and returns tokens
func (p *LiteParser) Parse(input []byte) ([]Token, error) {
	// Reset parser state
	p.input = input
	p.pos = 0
	p.tokens = p.tokens[:0]
	p.stack = p.stack[:0]

	// Parse the root value
	if err := p.parseValue(); err != nil {
		return nil, err
	}

	// Skip trailing whitespace
	p.skipWhitespace()

	// Ensure we've consumed all input
	if p.pos < len(p.input) {
		return nil, fmt.Errorf("unexpected content after JSON at position %d", p.pos)
	}

	return p.tokens, nil
}

// Get parent token index (-1 if none)
func (p *LiteParser) parent() int {
	if len(p.stack) > 0 {
		return p.stack[len(p.stack)-1]
	}
	return -1
}

// Add a token and return its index
func (p *LiteParser) addToken(typ TokenType, start, end int) int {
	idx := len(p.tokens)
	p.tokens = append(p.tokens, Token{
		Type:   typ,
		Start:  start,
		End:    end,
		Parent: p.parent(),
	})
	return idx
}

// Push token index onto stack
func (p *LiteParser) push(idx int) {
	p.stack = append(p.stack, idx)
}

// Pop token index from stack
func (p *LiteParser) pop() {
	if len(p.stack) > 0 {
		p.stack = p.stack[:len(p.stack)-1]
	}
}

// Skip whitespace characters
func (p *LiteParser) skipWhitespace() {
	for p.pos < len(p.input) {
		switch p.input[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

// Parse a JSON value
func (p *LiteParser) parseValue() error {
	p.skipWhitespace()

	if p.pos >= len(p.input) {
		return fmt.Errorf("unexpected end of input")
	}

	switch p.input[p.pos] {
	case '{':
		return p.parseObject()
	case '[':
		return p.parseArray()
	case '"':
		return p.parseString()
	case 't':
		return p.parseLiteral("true", TokenTrue)
	case 'f':
		return p.parseLiteral("false", TokenFalse)
	case 'n':
		return p.parseLiteral("null", TokenNull)
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return p.parseNumber()
	default:
		return fmt.Errorf("unexpected character '%c' at position %d", p.input[p.pos], p.pos)
	}
}

// Parse an object
func (p *LiteParser) parseObject() error {
	start := p.pos
	p.pos++ // skip '{'

	idx := p.addToken(TokenObjectStart, start, p.pos)
	p.push(idx)

	first := true
	for {
		p.skipWhitespace()

		if p.pos >= len(p.input) {
			return fmt.Errorf("unterminated object")
		}

		// Check for empty object or end
		if p.input[p.pos] == '}' {
			p.addToken(TokenObjectEnd, p.pos, p.pos+1)
			p.pos++
			p.pop()
			return nil
		}

		// Expect comma between elements
		if !first {
			if p.input[p.pos] != ',' {
				return fmt.Errorf("expected ',' at position %d", p.pos)
			}
			p.addToken(TokenComma, p.pos, p.pos+1)
			p.pos++
			p.skipWhitespace()
		}
		first = false

		// Parse key (must be string)
		if p.pos >= len(p.input) || p.input[p.pos] != '"' {
			return fmt.Errorf("expected string key at position %d", p.pos)
		}
		if err := p.parseString(); err != nil {
			return err
		}

		// Parse colon
		p.skipWhitespace()
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

// Parse an array
func (p *LiteParser) parseArray() error {
	start := p.pos
	p.pos++ // skip '['

	idx := p.addToken(TokenArrayStart, start, p.pos)
	p.push(idx)

	first := true
	for {
		p.skipWhitespace()

		if p.pos >= len(p.input) {
			return fmt.Errorf("unterminated array")
		}

		// Check for empty array or end
		if p.input[p.pos] == ']' {
			p.addToken(TokenArrayEnd, p.pos, p.pos+1)
			p.pos++
			p.pop()
			return nil
		}

		// Expect comma between elements
		if !first {
			if p.input[p.pos] != ',' {
				return fmt.Errorf("expected ',' at position %d", p.pos)
			}
			p.addToken(TokenComma, p.pos, p.pos+1)
			p.pos++
			p.skipWhitespace()
		}
		first = false

		// Parse value
		if err := p.parseValue(); err != nil {
			return err
		}
	}
}

// Parse a string
func (p *LiteParser) parseString() error {
	start := p.pos
	p.pos++ // skip opening '"'

	for p.pos < len(p.input) {
		switch p.input[p.pos] {
		case '"':
			// End of string
			p.pos++
			p.addToken(TokenString, start, p.pos)
			return nil
		case '\\':
			// Skip escaped character
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

// Parse a number
func (p *LiteParser) parseNumber() error {
	start := p.pos

	// Optional minus
	if p.input[p.pos] == '-' {
		p.pos++
		if p.pos >= len(p.input) {
			return fmt.Errorf("invalid number")
		}
	}

	// Integer part
	if p.pos >= len(p.input) || !isDigit(p.input[p.pos]) {
		return fmt.Errorf("invalid number")
	}

	if p.input[p.pos] == '0' {
		// Leading zero - must be followed by . or e/E or end
		p.pos++
	} else {
		// One or more digits
		for p.pos < len(p.input) && isDigit(p.input[p.pos]) {
			p.pos++
		}
	}

	// Fractional part
	if p.pos < len(p.input) && p.input[p.pos] == '.' {
		p.pos++
		if p.pos >= len(p.input) || !isDigit(p.input[p.pos]) {
			return fmt.Errorf("invalid number: no digits after decimal point")
		}
		for p.pos < len(p.input) && isDigit(p.input[p.pos]) {
			p.pos++
		}
	}

	// Exponent part
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

// Parse a literal (true, false, null)
func (p *LiteParser) parseLiteral(literal string, tokenType TokenType) error {
	start := p.pos

	// Check if we have enough characters
	if p.pos+len(literal) > len(p.input) {
		return fmt.Errorf("unexpected end of input")
	}

	// Check if the literal matches
	for i := 0; i < len(literal); i++ {
		if p.input[p.pos+i] != literal[i] {
			return fmt.Errorf("invalid literal at position %d", p.pos)
		}
	}

	p.pos += len(literal)
	p.addToken(tokenType, start, p.pos)
	return nil
}

// Helper function to check if byte is digit
func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// GetTokenValue returns the string value of a token
func (p *LiteParser) GetTokenValue(token Token) string {
	if token.Start < len(p.input) && token.End <= len(p.input) {
		return string(p.input[token.Start:token.End])
	}
	return ""
}

// ============================================================================
// BENCHMARKS
// ============================================================================

// Test data
var (
	tinyJSON   = []byte(`{"ok":true}`)
	smallJSON  = []byte(`{"id":123,"name":"test","email":"test@example.com","active":true}`)
	mediumJSON = []byte(`{"users":[` + strings.Repeat(`{"id":1,"name":"John Doe","email":"john@example.com","active":true},`, 100) + `]}`)
	largeJSON  = []byte(`{"data":[` + strings.Repeat(`{"id":1,"value":123.45,"nested":{"a":1,"b":2,"c":3},"tags":["x","y","z"]},`, 1000) + `]}`)
)

// Global vars to prevent compiler optimizations
var (
	liteTokens []Token
	liteError  error
	jsonResult interface{}
)

func BenchmarkLite(b *testing.B) {
	cases := []struct {
		name string
		data []byte
	}{
		{"Tiny", tinyJSON},
		{"Small", smallJSON},
		{"Medium", mediumJSON},
		{"Large", largeJSON},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(tc.data)))
			b.ReportAllocs()
			b.ResetTimer()

			parser := NewLiteParser()
			for i := 0; i < b.N; i++ {
				liteTokens, liteError = parser.Parse(tc.data)
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
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(tc.data)))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				liteError = json.Unmarshal(tc.data, &jsonResult)
			}
		})
	}
}

func BenchmarkLiteReuse(b *testing.B) {
	// Benchmark with parser reuse (simulating a server scenario)
	cases := []struct {
		name string
		data []byte
	}{
		{"Small", smallJSON},
		{"Medium", mediumJSON},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(tc.data)))
			b.ReportAllocs()

			// Create parser once and reuse
			parser := NewLiteParser()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				liteTokens, liteError = parser.Parse(tc.data)
			}
		})
	}
}

// ============================================================================
// VALIDATION TESTS
// ============================================================================

func TestLiteParser(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		minTokens int
	}{
		{
			name:      "empty object",
			input:     "{}",
			wantErr:   false,
			minTokens: 2, // { }
		},
		{
			name:      "empty array",
			input:     "[]",
			wantErr:   false,
			minTokens: 2, // [ ]
		},
		{
			name:      "simple object",
			input:     `{"key":"value"}`,
			wantErr:   false,
			minTokens: 5, // { "key" : "value" }
		},
		{
			name:      "number types",
			input:     `[123, -456, 0.789, 1.23e4, -5.67E-8]`,
			wantErr:   false,
			minTokens: 11, // [ num , num , num , num , num ]
		},
		{
			name:      "literals",
			input:     `[true, false, null]`,
			wantErr:   false,
			minTokens: 7, // [ true , false , null ]
		},
		{
			name:      "nested",
			input:     `{"a":{"b":{"c":[1,2,3]}}}`,
			wantErr:   false,
			minTokens: 17,
		},
		{
			name:    "invalid json",
			input:   `{"key":}`,
			wantErr: true,
		},
		{
			name:    "trailing comma",
			input:   `[1,2,3,]`,
			wantErr: true,
		},
	}

	parser := NewLiteParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := parser.Parse([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(tokens) < tt.minTokens {
				t.Errorf("Parse() returned %d tokens, want at least %d", len(tokens), tt.minTokens)
			}
		})
	}
}

// ============================================================================
// USAGE EXAMPLE
// ============================================================================

func ExampleLiteParser() {
	input := []byte(`{
		"name": "John Doe",
		"age": 30,
		"active": true,
		"tags": ["go", "json", "parser"]
	}`)

	parser := NewLiteParser()
	tokens, err := parser.Parse(input)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
		return
	}

	fmt.Printf("Found %d tokens\n", len(tokens))

	// Print all string tokens
	for _, token := range tokens {
		if token.Type == TokenString {
			fmt.Printf("String: %s\n", parser.GetTokenValue(token))
		}
	}

	// Output:
	// Found 23 tokens
	// String: "name"
	// String: "John Doe"
	// String: "age"
	// String: "active"
	// String: "tags"
	// String: "go"
	// String: "json"
	// String: "parser"
}
