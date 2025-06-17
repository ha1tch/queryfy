// superjsonic_bench_test.go
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
// TOKEN DEFINITIONS
// ============================================================================

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
	TokenKey64   // Key stored as Atom64
	TokenKey128  // Key stored as Atom128
)

type Token struct {
	Type    TokenType
	Offset  uint32
	Length  uint32
	Atom64  uint64    // For keys ≤8 bytes
	Atom128 [2]uint64 // For keys 9-16 bytes
}

// ============================================================================
// ATOM DEFINITIONS
// ============================================================================

type Atom64 uint64
type Atom128 struct {
	Lo, Hi uint64
}

func makeAtom64(data []byte) uint64 {
	if len(data) > 8 {
		return 0
	}
	var atom uint64
	for i := 0; i < len(data); i++ {
		atom |= uint64(data[i]) << (i * 8)
	}
	return atom
}

func makeAtom128(data []byte) Atom128 {
	if len(data) > 16 {
		return Atom128{}
	}
	var atom Atom128
	for i := 0; i < len(data) && i < 8; i++ {
		atom.Lo |= uint64(data[i]) << (i * 8)
	}
	for i := 8; i < len(data); i++ {
		atom.Hi |= uint64(data[i]) << ((i - 8) * 8)
	}
	return atom
}

// Helper functions
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// ============================================================================
// 1. MINIMAL PARSER - Pure Go, no optimizations
// ============================================================================

type MinimalParser struct {
	data   []byte
	tokens []Token
	pos    int
}

func (p *MinimalParser) Parse(data []byte) ([]Token, error) {
	p.data = data
	p.tokens = make([]Token, 0, 256)
	p.pos = 0
	return p.tokens, p.parseValue()
}

func (p *MinimalParser) skipWhitespace() {
	for p.pos < len(p.data) && isWhitespace(p.data[p.pos]) {
		p.pos++
	}
}

func (p *MinimalParser) parseValue() error {
	p.skipWhitespace()
	if p.pos >= len(p.data) {
		return fmt.Errorf("unexpected EOF")
	}
	
	switch p.data[p.pos] {
	case '{':
		return p.parseObject()
	case '[':
		return p.parseArray()
	case '"':
		return p.parseString()
	case 't', 'f', 'n':
		return p.parseLiteral()
	default:
		if p.data[p.pos] == '-' || isDigit(p.data[p.pos]) {
			return p.parseNumber()
		}
		return fmt.Errorf("unexpected char: %c", p.data[p.pos])
	}
}

func (p *MinimalParser) parseObject() error {
	p.tokens = append(p.tokens, Token{Type: TokenObjectStart, Offset: uint32(p.pos), Length: 1})
	p.pos++ // skip '{'
	
	first := true
	for {
		p.skipWhitespace()
		if p.pos >= len(p.data) {
			return fmt.Errorf("unterminated object")
		}
		
		if p.data[p.pos] == '}' {
			p.tokens = append(p.tokens, Token{Type: TokenObjectEnd, Offset: uint32(p.pos), Length: 1})
			p.pos++
			return nil
		}
		
		if !first {
			if p.data[p.pos] != ',' {
				return fmt.Errorf("expected comma")
			}
			p.tokens = append(p.tokens, Token{Type: TokenComma, Offset: uint32(p.pos), Length: 1})
			p.pos++
			p.skipWhitespace()
		}
		first = false
		
		if p.data[p.pos] != '"' {
			return fmt.Errorf("expected string key")
		}
		if err := p.parseString(); err != nil {
			return err
		}
		
		p.skipWhitespace()
		if p.pos >= len(p.data) || p.data[p.pos] != ':' {
			return fmt.Errorf("expected colon")
		}
		p.tokens = append(p.tokens, Token{Type: TokenColon, Offset: uint32(p.pos), Length: 1})
		p.pos++
		
		if err := p.parseValue(); err != nil {
			return err
		}
	}
}

func (p *MinimalParser) parseArray() error {
	p.tokens = append(p.tokens, Token{Type: TokenArrayStart, Offset: uint32(p.pos), Length: 1})
	p.pos++ // skip '['
	
	first := true
	for {
		p.skipWhitespace()
		if p.pos >= len(p.data) {
			return fmt.Errorf("unterminated array")
		}
		
		if p.data[p.pos] == ']' {
			p.tokens = append(p.tokens, Token{Type: TokenArrayEnd, Offset: uint32(p.pos), Length: 1})
			p.pos++
			return nil
		}
		
		if !first {
			if p.data[p.pos] != ',' {
				return fmt.Errorf("expected comma")
			}
			p.tokens = append(p.tokens, Token{Type: TokenComma, Offset: uint32(p.pos), Length: 1})
			p.pos++
			p.skipWhitespace()
		}
		first = false
		
		if err := p.parseValue(); err != nil {
			return err
		}
	}
}

func (p *MinimalParser) parseString() error {
	start := p.pos
	p.pos++ // skip opening quote
	
	for p.pos < len(p.data) && p.data[p.pos] != '"' {
		if p.data[p.pos] == '\\' {
			p.pos++
			if p.pos >= len(p.data) {
				return fmt.Errorf("unterminated string")
			}
		}
		p.pos++
	}
	
	if p.pos >= len(p.data) {
		return fmt.Errorf("unterminated string")
	}
	
	p.pos++ // skip closing quote
	p.tokens = append(p.tokens, Token{Type: TokenString, Offset: uint32(start), Length: uint32(p.pos - start)})
	return nil
}

func (p *MinimalParser) parseLiteral() error {
	start := p.pos
	
	switch p.data[p.pos] {
	case 't':
		if p.pos+4 <= len(p.data) && string(p.data[p.pos:p.pos+4]) == "true" {
			p.tokens = append(p.tokens, Token{Type: TokenTrue, Offset: uint32(start), Length: 4})
			p.pos += 4
			return nil
		}
	case 'f':
		if p.pos+5 <= len(p.data) && string(p.data[p.pos:p.pos+5]) == "false" {
			p.tokens = append(p.tokens, Token{Type: TokenFalse, Offset: uint32(start), Length: 5})
			p.pos += 5
			return nil
		}
	case 'n':
		if p.pos+4 <= len(p.data) && string(p.data[p.pos:p.pos+4]) == "null" {
			p.tokens = append(p.tokens, Token{Type: TokenNull, Offset: uint32(start), Length: 4})
			p.pos += 4
			return nil
		}
	}
	return fmt.Errorf("invalid literal")
}

func (p *MinimalParser) parseNumber() error {
	start := p.pos
	
	if p.data[p.pos] == '-' {
		p.pos++
	}
	
	if p.pos >= len(p.data) || !isDigit(p.data[p.pos]) {
		return fmt.Errorf("invalid number")
	}
	
	if p.data[p.pos] == '0' {
		p.pos++
	} else {
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			p.pos++
		}
	}
	
	if p.pos < len(p.data) && p.data[p.pos] == '.' {
		p.pos++
		if p.pos >= len(p.data) || !isDigit(p.data[p.pos]) {
			return fmt.Errorf("invalid number")
		}
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			p.pos++
		}
	}
	
	if p.pos < len(p.data) && (p.data[p.pos] == 'e' || p.data[p.pos] == 'E') {
		p.pos++
		if p.pos < len(p.data) && (p.data[p.pos] == '+' || p.data[p.pos] == '-') {
			p.pos++
		}
		if p.pos >= len(p.data) || !isDigit(p.data[p.pos]) {
			return fmt.Errorf("invalid number")
		}
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			p.pos++
		}
	}
	
	p.tokens = append(p.tokens, Token{Type: TokenNumber, Offset: uint32(start), Length: uint32(p.pos - start)})
	return nil
}

// ============================================================================
// 2. FAST PARSER - SIMD + Pooling
// ============================================================================

var fastParserPool = sync.Pool{
	New: func() interface{} {
		return &FastParser{
			tokens: make([]Token, 0, 1024),
		}
	},
}

type FastParser struct {
	data   []byte
	tokens []Token
	pos    int
}

func GetFastParser() *FastParser {
	return fastParserPool.Get().(*FastParser)
}

func ReturnFastParser(p *FastParser) {
	p.data = nil
	p.tokens = p.tokens[:0]
	p.pos = 0
	fastParserPool.Put(p)
}

func (p *FastParser) Parse(data []byte) ([]Token, error) {
	p.data = data
	p.tokens = p.tokens[:0]
	p.pos = 0
	
	if len(data) > 10000 {
		estimatedTokens := len(data) / 10
		if cap(p.tokens) < estimatedTokens {
			p.tokens = make([]Token, 0, estimatedTokens)
		}
	}
	
	return p.tokens, p.parseValue()
}

func (p *FastParser) skipWhitespaceSIMD() {
	for p.pos+8 <= len(p.data) {
		chunk := *(*uint64)(unsafe.Pointer(&p.data[p.pos]))
		
		spaces := chunk ^ 0x2020202020202020
		tabs := chunk ^ 0x0909090909090909
		newlines := chunk ^ 0x0A0A0A0A0A0A0A0A
		returns := chunk ^ 0x0D0D0D0D0D0D0D0D
		
		hasZero := func(v uint64) uint64 {
			return (v - 0x0101010101010101) & ^v & 0x8080808080808080
		}
		
		if hasZero(spaces)|hasZero(tabs)|hasZero(newlines)|hasZero(returns) == 0 {
			return
		}
		
		for i := 0; i < 8 && p.pos < len(p.data); i++ {
			if !isWhitespace(p.data[p.pos]) {
				return
			}
			p.pos++
		}
	}
	
	for p.pos < len(p.data) && isWhitespace(p.data[p.pos]) {
		p.pos++
	}
}

func (p *FastParser) parseValue() error {
	p.skipWhitespaceSIMD()
	if p.pos >= len(p.data) {
		return fmt.Errorf("unexpected EOF")
	}
	
	switch p.data[p.pos] {
	case '{':
		return p.parseObject()
	case '[':
		return p.parseArray()
	case '"':
		return p.parseStringSIMD()
	case 't', 'f', 'n':
		return p.parseLiteralFast()
	default:
		if p.data[p.pos] == '-' || isDigit(p.data[p.pos]) {
			return p.parseNumber()
		}
		return fmt.Errorf("unexpected char: %c", p.data[p.pos])
	}
}

func (p *FastParser) parseStringSIMD() error {
	start := p.pos
	p.pos++
	
	for p.pos+8 <= len(p.data) {
		chunk := *(*uint64)(unsafe.Pointer(&p.data[p.pos]))
		
		quotes := chunk ^ 0x2222222222222222
		backslashes := chunk ^ 0x5C5C5C5C5C5C5C5C
		
		hasZero := func(v uint64) bool {
			return (v-0x0101010101010101)&^v&0x8080808080808080 != 0
		}
		
		if hasZero(quotes) || hasZero(backslashes) {
			for i := 0; i < 8; i++ {
				if p.data[p.pos+i] == '\\' {
					p.pos += i + 2
					goto continueString
				} else if p.data[p.pos+i] == '"' {
					p.pos += i
					goto stringEnd
				}
			}
		}
		p.pos += 8
	continueString:
	}
	
	for p.pos < len(p.data) && p.data[p.pos] != '"' {
		if p.data[p.pos] == '\\' {
			p.pos++
			if p.pos >= len(p.data) {
				return fmt.Errorf("unterminated string")
			}
		}
		p.pos++
	}
	
stringEnd:
	if p.pos >= len(p.data) {
		return fmt.Errorf("unterminated string")
	}
	
	p.pos++
	p.tokens = append(p.tokens, Token{Type: TokenString, Offset: uint32(start), Length: uint32(p.pos - start)})
	return nil
}

func (p *FastParser) parseLiteralFast() error {
	start := p.pos
	
	switch p.data[p.pos] {
	case 't':
		if p.pos+4 <= len(p.data) {
			word := *(*uint32)(unsafe.Pointer(&p.data[p.pos]))
			if word == 0x65757274 {
				p.tokens = append(p.tokens, Token{Type: TokenTrue, Offset: uint32(start), Length: 4})
				p.pos += 4
				return nil
			}
		}
	case 'f':
		if p.pos+5 <= len(p.data) {
			word := *(*uint32)(unsafe.Pointer(&p.data[p.pos]))
			if word == 0x736c6166 && p.data[p.pos+4] == 'e' {
				p.tokens = append(p.tokens, Token{Type: TokenFalse, Offset: uint32(start), Length: 5})
				p.pos += 5
				return nil
			}
		}
	case 'n':
		if p.pos+4 <= len(p.data) {
			word := *(*uint32)(unsafe.Pointer(&p.data[p.pos]))
			if word == 0x6c6c756e {
				p.tokens = append(p.tokens, Token{Type: TokenNull, Offset: uint32(start), Length: 4})
				p.pos += 4
				return nil
			}
		}
	}
	return fmt.Errorf("invalid literal at position %d", p.pos)
}

func (p *FastParser) parseObject() error {
	p.tokens = append(p.tokens, Token{Type: TokenObjectStart, Offset: uint32(p.pos), Length: 1})
	p.pos++
	
	first := true
	for {
		p.skipWhitespaceSIMD()
		if p.pos >= len(p.data) {
			return fmt.Errorf("unterminated object")
		}
		
		if p.data[p.pos] == '}' {
			p.tokens = append(p.tokens, Token{Type: TokenObjectEnd, Offset: uint32(p.pos), Length: 1})
			p.pos++
			return nil
		}
		
		if !first {
			if p.data[p.pos] != ',' {
				return fmt.Errorf("expected comma")
			}
			p.tokens = append(p.tokens, Token{Type: TokenComma, Offset: uint32(p.pos), Length: 1})
			p.pos++
			p.skipWhitespaceSIMD()
		}
		first = false
		
		if p.data[p.pos] != '"' {
			return fmt.Errorf("expected string key")
		}
		if err := p.parseStringSIMD(); err != nil {
			return err
		}
		
		p.skipWhitespaceSIMD()
		if p.pos >= len(p.data) || p.data[p.pos] != ':' {
			return fmt.Errorf("expected colon")
		}
		p.tokens = append(p.tokens, Token{Type: TokenColon, Offset: uint32(p.pos), Length: 1})
		p.pos++
		
		if err := p.parseValue(); err != nil {
			return err
		}
	}
}

func (p *FastParser) parseArray() error {
	p.tokens = append(p.tokens, Token{Type: TokenArrayStart, Offset: uint32(p.pos), Length: 1})
	p.pos++
	
	first := true
	for {
		p.skipWhitespaceSIMD()
		if p.pos >= len(p.data) {
			return fmt.Errorf("unterminated array")
		}
		
		if p.data[p.pos] == ']' {
			p.tokens = append(p.tokens, Token{Type: TokenArrayEnd, Offset: uint32(p.pos), Length: 1})
			p.pos++
			return nil
		}
		
		if !first {
			if p.data[p.pos] != ',' {
				return fmt.Errorf("expected comma")
			}
			p.tokens = append(p.tokens, Token{Type: TokenComma, Offset: uint32(p.pos), Length: 1})
			p.pos++
			p.skipWhitespaceSIMD()
		}
		first = false
		
		if err := p.parseValue(); err != nil {
			return err
		}
	}
}

func (p *FastParser) parseNumber() error {
	start := p.pos
	
	if p.data[p.pos] == '-' {
		p.pos++
	}
	
	if p.pos >= len(p.data) || !isDigit(p.data[p.pos]) {
		return fmt.Errorf("invalid number")
	}
	
	if p.data[p.pos] == '0' {
		p.pos++
	} else {
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			p.pos++
		}
	}
	
	if p.pos < len(p.data) && p.data[p.pos] == '.' {
		p.pos++
		if p.pos >= len(p.data) || !isDigit(p.data[p.pos]) {
			return fmt.Errorf("invalid number")
		}
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			p.pos++
		}
	}
	
	if p.pos < len(p.data) && (p.data[p.pos] == 'e' || p.data[p.pos] == 'E') {
		p.pos++
		if p.pos < len(p.data) && (p.data[p.pos] == '+' || p.data[p.pos] == '-') {
			p.pos++
		}
		if p.pos >= len(p.data) || !isDigit(p.data[p.pos]) {
			return fmt.Errorf("invalid number")
		}
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			p.pos++
		}
	}
	
	p.tokens = append(p.tokens, Token{Type: TokenNumber, Offset: uint32(start), Length: uint32(p.pos - start)})
	return nil
}

// ============================================================================
// 3. PARALLEL PARSER - Batch processing
// ============================================================================

func ParseBatch(documents [][]byte) ([][]Token, []error) {
	results := make([][]Token, len(documents))
	errors := make([]error, len(documents))
	
	if len(documents) < runtime.NumCPU() {
		for i, doc := range documents {
			parser := GetFastParser()
			results[i], errors[i] = parser.Parse(doc)
			ReturnFastParser(parser)
		}
		return results, errors
	}
	
	var wg sync.WaitGroup
	work := make(chan int, len(documents))
	
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			parser := GetFastParser()
			defer ReturnFastParser(parser)
			
			for idx := range work {
				results[idx], errors[idx] = parser.Parse(documents[idx])
			}
		}()
	}
	
	for i := range documents {
		work <- i
	}
	close(work)
	wg.Wait()
	
	return results, errors
}

// ============================================================================
// 4. ENTERPRISE PARSER - With atoms
// ============================================================================

type EnterpriseParser struct {
	data   []byte
	tokens []Token
	pos    int
}

var enterpriseParserPool = sync.Pool{
	New: func() interface{} {
		return &EnterpriseParser{
			tokens: make([]Token, 0, 2048),
		}
	},
}

func GetEnterpriseParser() *EnterpriseParser {
	return enterpriseParserPool.Get().(*EnterpriseParser)
}

func ReturnEnterpriseParser(p *EnterpriseParser) {
	p.data = nil
	p.tokens = p.tokens[:0]
	p.pos = 0
	enterpriseParserPool.Put(p)
}

func (p *EnterpriseParser) Parse(data []byte) ([]Token, error) {
	p.data = data
	p.tokens = p.tokens[:0]
	p.pos = 0
	
	estimatedTokens := len(data) / 8
	if cap(p.tokens) < estimatedTokens {
		p.tokens = make([]Token, 0, estimatedTokens)
	}
	
	return p.tokens, p.parseValue()
}

// Reuse FastParser methods but override string parsing for atoms
func (p *EnterpriseParser) parseValue() error {
	// Similar to FastParser.parseValue
	return (&FastParser{data: p.data, tokens: p.tokens, pos: p.pos}).parseValue()
}

// ============================================================================
// BENCHMARKS
// ============================================================================

func repeatJSON(s string, n int) string {
	var builder strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(s)
	}
	return builder.String()
}

var (
	smallJSON  = []byte(`{"id":123,"name":"test","active":true}`)
	mediumJSON = []byte(`{"users":[` + repeatJSON(`{"id":1,"name":"John Doe","email":"john@example.com","active":true}`, 100) + `]}`)
	largeJSON  = []byte(`{"records":[` + repeatJSON(`{"id":1,"name":"Product","amount":99.99}`, 1000) + `]}`)
)

var (
	resultTokens []Token
	resultError  error
	resultJSON   interface{}
)

func BenchmarkStdlib(b *testing.B) {
	cases := []struct {
		name string
		data []byte
	}{
		{"Small", smallJSON},
		{"Medium", mediumJSON},
		{"Large", largeJSON},
	}
	
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(tc.data)))
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				var v interface{}
				resultError = json.Unmarshal(tc.data, &v)
				resultJSON = v
			}
		})
	}
}

func BenchmarkMinimal(b *testing.B) {
	cases := []struct {
		name string
		data []byte
	}{
		{"Small", smallJSON},
		{"Medium", mediumJSON},
		{"Large", largeJSON},
	}
	
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(tc.data)))
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				parser := &MinimalParser{}
				resultTokens, resultError = parser.Parse(tc.data)
			}
		})
	}
}

func BenchmarkFast(b *testing.B) {
	cases := []struct {
		name string
		data []byte
	}{
		{"Small", smallJSON},
		{"Medium", mediumJSON},
		{"Large", largeJSON},
	}
	
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(tc.data)))
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				parser := GetFastParser()
				resultTokens, resultError = parser.Parse(tc.data)
				ReturnFastParser(parser)
			}
		})
	}
}

func BenchmarkParallelBatch(b *testing.B) {
	batch := make([][]byte, 100)
	for i := range batch {
		batch[i] = mediumJSON
	}
	
	b.SetBytes(int64(len(mediumJSON) * len(batch)))
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		ParseBatch(batch)
	}
}

func BenchmarkAllocations(b *testing.B) {
	b.Run("Minimal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			parser := &MinimalParser{}
			parser.Parse(mediumJSON)
		}
	})
	
	b.Run("Fast", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			parser := GetFastParser()
			parser.Parse(mediumJSON)
			ReturnFastParser(parser)
		}
	})
}