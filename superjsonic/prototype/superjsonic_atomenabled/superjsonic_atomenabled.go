package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"unsafe"
)

// ============================================================================
// ATOM DEFINITIONS
// ============================================================================

type Atom64 uint64
type Atom128 struct {
	Lo uint64
	Hi uint64
}

// Common Atom64 constants - covering most frequent JSON keys
const (
	// 2-4 byte atoms
	Atom64_Id    Atom64 = 0x6469       // "id"
	Atom64_Key   Atom64 = 0x79656b     // "key"
	Atom64_Name  Atom64 = 0x656d616e   // "name"
	Atom64_Type  Atom64 = 0x65707974   // "type"
	Atom64_Data  Atom64 = 0x61746164   // "data"
	Atom64_Code  Atom64 = 0x65646f63   // "code"
	Atom64_Text  Atom64 = 0x74786574   // "text"
	Atom64_Value Atom64 = 0x65756c6176 // "value"

	// 5-8 byte atoms
	Atom64_Email    Atom64 = 0x6c69616d65       // "email"
	Atom64_Status   Atom64 = 0x737574617473     // "status"
	Atom64_Error    Atom64 = 0x726f727265       // "error"
	Atom64_Items    Atom64 = 0x736d657469       // "items"
	Atom64_Count    Atom64 = 0x746e756f63       // "count"
	Atom64_Total    Atom64 = 0x6c61746f74       // "total"
	Atom64_Price    Atom64 = 0x6563697270       // "price"
	Atom64_Amount   Atom64 = 0x746e756f6d61     // "amount"
	Atom64_Street   Atom64 = 0x746565727473     // "street"
	Atom64_City     Atom64 = 0x79746963         // "city"
	Atom64_Country  Atom64 = 0x7972746e756f63   // "country" (7)
	Atom64_Created  Atom64 = 0x6465746165726308 // "created" (7)
	Atom64_Updated  Atom64 = 0x6465746164707508 // "updated" (7)
	Atom64_Customer Atom64 = 0x72656d6f74737563 // "customer" (8)
	Atom64_Password Atom64 = 0x64726f7773736170 // "password" (8)
)

// Common Atom128 constants - for longer frequent keys
var (
	// Payment related (9-16 bytes)
	Atom128_PaymentMethod  = makeAtom128Static("paymentMethod")  // 13 bytes
	Atom128_CardNumber     = makeAtom128Static("cardNumber")     // 10 bytes
	Atom128_SecurityCode   = makeAtom128Static("securityCode")   // 12 bytes
	Atom128_ExpirationDate = makeAtom128Static("expirationDate") // 14 bytes

	// Document related
	Atom128_DocumentType   = makeAtom128Static("documentType")   // 12 bytes
	Atom128_DocumentNumber = makeAtom128Static("documentNumber") // 14 bytes

	// Other common
	Atom128_Subscriptions  = makeAtom128Static("subscriptions")  // 13 bytes
	Atom128_BillingAccount = makeAtom128Static("billingAccount") // 14 bytes
)

// Atom creation helpers
func makeAtom64(data []byte) Atom64 {
	if len(data) > 8 {
		return 0
	}
	var atom uint64
	for i := 0; i < len(data); i++ {
		atom |= uint64(data[i]) << (i * 8)
	}
	return Atom64(atom)
}

func makeAtom128(data []byte) Atom128 {
	if len(data) > 16 {
		return Atom128{}
	}

	var atom Atom128
	// Fill low 8 bytes
	for i := 0; i < len(data) && i < 8; i++ {
		atom.Lo |= uint64(data[i]) << (i * 8)
	}
	// Fill high 8 bytes
	for i := 8; i < len(data) && i < 16; i++ {
		atom.Hi |= uint64(data[i-8]) << ((i - 8) * 8)
	}
	return atom
}

func makeAtom128Static(s string) Atom128 {
	return makeAtom128([]byte(s))
}

// ============================================================================
// ENHANCED TOKEN SYSTEM WITH ATOMS
// ============================================================================

type TokenType uint8

const (
	// Basic tokens
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

	// Atom tokens
	TokenKey64  // Key stored as Atom64
	TokenKey128 // Key stored as Atom128
)

// Token with atom support
type Token struct {
	Type    TokenType
	Offset  uint32  // For regular strings
	Length  uint32  // For regular strings
	Atom64  Atom64  // For keys ≤8 bytes
	Atom128 Atom128 // For keys 9-16 bytes
}

// Token methods
func (t Token) IsAtom64() bool {
	return t.Type == TokenKey64
}

func (t Token) IsAtom128() bool {
	return t.Type == TokenKey128
}

// ============================================================================
// SIMD OPERATIONS (from existing code)
// ============================================================================

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

// ============================================================================
// ENHANCED PARSER WITH ATOM SUPPORT
// ============================================================================

type ArrayInfo struct {
	isLargeArray      bool
	tokensPerElement  int
	estimatedElements int
	firstElementEnd   int
}

type FastJSONParser struct {
	data      []byte
	tokens    []Token
	pos       int
	simd      SimdOps
	arrayInfo ArrayInfo

	// Atom caching for frequent keys
	atomCache map[string]interface{} // string -> Atom64 or Atom128
	inObject  bool                   // Track if we're parsing object keys
	expectKey bool                   // Track if next string is a key
}

// Parser pool
var parserPool = sync.Pool{
	New: func() interface{} {
		return &FastJSONParser{
			tokens:    make([]Token, 0, 1024),
			simd:      SimdOps{},
			atomCache: make(map[string]interface{}, 128),
		}
	},
}

func GetParser() *FastJSONParser {
	return parserPool.Get().(*FastJSONParser)
}

func ReturnParser(p *FastJSONParser) {
	p.data = nil
	p.tokens = p.tokens[:0]
	p.pos = 0
	p.inObject = false
	p.expectKey = false
	// Keep atomCache - it's beneficial across parses
	parserPool.Put(p)
}

// Main parse function with atom support
func (p *FastJSONParser) Parse(jsonData []byte) error {
	p.data = jsonData
	p.tokens = p.tokens[:0]
	p.pos = 0
	p.inObject = false
	p.expectKey = false

	// Array detection
	p.arrayInfo = p.detectLargeArray(jsonData)
	if p.arrayInfo.isLargeArray && cap(p.tokens) < p.arrayInfo.estimatedElements*p.arrayInfo.tokensPerElement {
		p.tokens = make([]Token, 0, p.arrayInfo.estimatedElements*p.arrayInfo.tokensPerElement)
	}

	return p.parseValue()
}

func (p *FastJSONParser) parseValue() error {
	p.skipWhitespace()
	if p.pos >= len(p.data) {
		return fmt.Errorf("unexpected end of JSON")
	}

	switch p.data[p.pos] {
	case '{':
		return p.parseObject()
	case '[':
		return p.parseArray()
	case '"':
		return p.parseString()
	case 't', 'f':
		return p.parseBool()
	case 'n':
		return p.parseNull()
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return p.parseNumber()
	default:
		return fmt.Errorf("unexpected character at position %d", p.pos)
	}
}

func (p *FastJSONParser) parseObject() error {
	p.addToken(TokenObjectStart, p.pos, 1, 0, Atom128{})
	p.pos++ // skip '{'
	p.inObject = true

	for {
		p.skipWhitespace()
		if p.pos >= len(p.data) {
			return fmt.Errorf("unterminated object")
		}

		if p.data[p.pos] == '}' {
			p.addToken(TokenObjectEnd, p.pos, 1, 0, Atom128{})
			p.pos++
			p.inObject = false
			return nil
		}

		// Parse key (with atom optimization)
		p.expectKey = true
		if err := p.parseString(); err != nil {
			return err
		}
		p.expectKey = false

		// Parse colon
		p.skipWhitespace()
		if p.pos >= len(p.data) || p.data[p.pos] != ':' {
			return fmt.Errorf("expected ':' after object key")
		}
		p.addToken(TokenColon, p.pos, 1, 0, Atom128{})
		p.pos++

		// Parse value
		if err := p.parseValue(); err != nil {
			return err
		}

		// Check for comma or end
		p.skipWhitespace()
		if p.pos >= len(p.data) {
			return fmt.Errorf("unterminated object")
		}

		if p.data[p.pos] == ',' {
			p.addToken(TokenComma, p.pos, 1, 0, Atom128{})
			p.pos++
		} else if p.data[p.pos] != '}' {
			return fmt.Errorf("expected ',' or '}' in object")
		}
	}
}

func (p *FastJSONParser) parseArray() error {
	p.addToken(TokenArrayStart, p.pos, 1, 0, Atom128{})
	p.pos++ // skip '['

	for {
		p.skipWhitespace()
		if p.pos >= len(p.data) {
			return fmt.Errorf("unterminated array")
		}

		if p.data[p.pos] == ']' {
			p.addToken(TokenArrayEnd, p.pos, 1, 0, Atom128{})
			p.pos++
			return nil
		}

		if err := p.parseValue(); err != nil {
			return err
		}

		p.skipWhitespace()
		if p.pos >= len(p.data) {
			return fmt.Errorf("unterminated array")
		}

		if p.data[p.pos] == ',' {
			p.addToken(TokenComma, p.pos, 1, 0, Atom128{})
			p.pos++
		} else if p.data[p.pos] != ']' {
			return fmt.Errorf("expected ',' or ']' in array")
		}
	}
}

func (p *FastJSONParser) parseString() error {
	start := p.pos
	p.pos++ // skip opening quote

	// Use SIMD for fast string scanning
	for p.pos+8 <= len(p.data) {
		v := *(*uint64)(unsafe.Pointer(&p.data[p.pos]))

		if p.simd.HasByte(v, '"') || p.simd.HasByte(v, '\\') {
			for j := 0; j < 8; j++ {
				if p.data[p.pos+j] == '\\' {
					p.pos += j + 2
					goto continueString
				} else if p.data[p.pos+j] == '"' {
					p.pos += j
					goto stringEnd
				}
			}
		}
		p.pos += 8
	continueString:
	}

	// Handle remaining bytes
	for p.pos < len(p.data) && p.data[p.pos] != '"' {
		if p.data[p.pos] == '\\' {
			p.pos++
		}
		p.pos++
	}

stringEnd:
	if p.pos >= len(p.data) {
		return fmt.Errorf("unterminated string")
	}

	// If this is an object key, try to make it an atom
	if p.expectKey && p.inObject {
		keyStart := start + 1 // skip opening quote
		keyLen := p.pos - keyStart

		if keyLen <= 8 {
			atom := p.makeAtom64(p.data[keyStart:p.pos])
			p.addToken(TokenKey64, 0, 0, atom, Atom128{})
		} else if keyLen <= 16 {
			atom := p.makeAtom128(p.data[keyStart:p.pos])
			p.addToken(TokenKey128, 0, 0, 0, atom)
		} else {
			// Regular string for long keys
			p.addToken(TokenString, start, p.pos-start+1, 0, Atom128{})
		}
	} else {
		// Regular string token
		p.addToken(TokenString, start, p.pos-start+1, 0, Atom128{})
	}

	p.pos++ // skip closing quote
	return nil
}

func (p *FastJSONParser) makeAtom64(data []byte) Atom64 {
	// Check cache first
	key := string(data)
	if cached, ok := p.atomCache[key]; ok {
		if atom, ok := cached.(Atom64); ok {
			return atom
		}
	}

	atom := makeAtom64(data)
	p.atomCache[key] = atom
	return atom
}

func (p *FastJSONParser) makeAtom128(data []byte) Atom128 {
	// Check cache first
	key := string(data)
	if cached, ok := p.atomCache[key]; ok {
		if atom, ok := cached.(Atom128); ok {
			return atom
		}
	}

	atom := makeAtom128(data)
	p.atomCache[key] = atom
	return atom
}

func (p *FastJSONParser) parseNumber() error {
	start := p.pos

	// Optional minus
	if p.data[p.pos] == '-' {
		p.pos++
	}

	// Digits
	for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
		p.pos++
	}

	// Optional decimal
	if p.pos < len(p.data) && p.data[p.pos] == '.' {
		p.pos++
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			p.pos++
		}
	}

	// Optional exponent
	if p.pos < len(p.data) && (p.data[p.pos] == 'e' || p.data[p.pos] == 'E') {
		p.pos++
		if p.pos < len(p.data) && (p.data[p.pos] == '+' || p.data[p.pos] == '-') {
			p.pos++
		}
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			p.pos++
		}
	}

	p.addToken(TokenNumber, start, p.pos-start, 0, Atom128{})
	return nil
}

func (p *FastJSONParser) parseBool() error {
	start := p.pos

	if p.pos+4 <= len(p.data) && string(p.data[p.pos:p.pos+4]) == "true" {
		p.addToken(TokenTrue, start, 4, 0, Atom128{})
		p.pos += 4
		return nil
	}

	if p.pos+5 <= len(p.data) && string(p.data[p.pos:p.pos+5]) == "false" {
		p.addToken(TokenFalse, start, 5, 0, Atom128{})
		p.pos += 5
		return nil
	}

	return fmt.Errorf("invalid boolean at position %d", p.pos)
}

func (p *FastJSONParser) parseNull() error {
	start := p.pos

	if p.pos+4 <= len(p.data) && string(p.data[p.pos:p.pos+4]) == "null" {
		p.addToken(TokenNull, start, 4, 0, Atom128{})
		p.pos += 4
		return nil
	}

	return fmt.Errorf("invalid null at position %d", p.pos)
}

func (p *FastJSONParser) skipWhitespace() {
	// SIMD-optimized whitespace skipping
	for p.pos+8 <= len(p.data) {
		v := *(*uint64)(unsafe.Pointer(&p.data[p.pos]))
		wsResult := p.simd.IsWhitespace(v)
		if wsResult == 0 {
			break
		}

		// Find first non-whitespace
		for j := 0; j < 8; j++ {
			if !isWhitespace(p.data[p.pos+j]) {
				p.pos += j
				return
			}
		}
		p.pos += 8
	}

	// Handle remaining bytes
	for p.pos < len(p.data) && isWhitespace(p.data[p.pos]) {
		p.pos++
	}
}

func (p *FastJSONParser) addToken(typ TokenType, offset, length int, atom64 Atom64, atom128 Atom128) {
	p.tokens = append(p.tokens, Token{
		Type:    typ,
		Offset:  uint32(offset),
		Length:  uint32(length),
		Atom64:  atom64,
		Atom128: atom128,
	})
}

// Array detection (from existing code)
func (p *FastJSONParser) detectLargeArray(data []byte) ArrayInfo {
	if len(data) < 100 {
		return ArrayInfo{}
	}

	// Check for common array patterns
	patterns := []string{
		`"items":[`,
		`"data":[`,
		`"results":[`,
		`"records":[`,
	}

	for _, pattern := range patterns {
		if idx := strings.Index(string(data[:min(200, len(data))]), pattern); idx >= 0 {
			// Estimate array size
			arrayStart := idx + len(pattern)
			elementSample := p.sampleArrayElement(data[arrayStart:])
			if elementSample > 0 {
				estimatedElements := (len(data) - arrayStart) / elementSample
				return ArrayInfo{
					isLargeArray:      true,
					tokensPerElement:  elementSample / 10, // rough estimate
					estimatedElements: estimatedElements,
				}
			}
		}
	}

	return ArrayInfo{}
}

func (p *FastJSONParser) sampleArrayElement(data []byte) int {
	depth := 0
	for i, b := range data {
		switch b {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
		if i > 1000 { // Don't sample too much
			break
		}
	}
	return 0
}

// Token access methods
func (p *FastJSONParser) GetTokens() []Token {
	return p.tokens
}

func (p *FastJSONParser) GetTokenValue(token Token) string {
	if token.IsAtom64() || token.IsAtom128() {
		// For atoms, we'd need reverse lookup - for now return placeholder
		return fmt.Sprintf("<atom:%d>", token.Type)
	}
	return string(p.data[token.Offset : token.Offset+token.Length])
}

// ============================================================================
// VALIDATION WITH ATOM SUPPORT
// ============================================================================

func ValidateWithAtoms(tokens []Token) error {
	for i, token := range tokens {
		switch {
		case token.IsAtom64():
			// Fast validation for known atoms
			switch token.Atom64 {
			case Atom64_Email:
				// Next token should be a valid email string
				if i+1 < len(tokens) && tokens[i+1].Type == TokenString {
					// Validate email format
				}
			case Atom64_Status:
				// Next token should be a valid status
				if i+1 < len(tokens) && tokens[i+1].Type == TokenString {
					// Validate status values
				}
			}

		case token.IsAtom128():
			// Validation for longer atoms
			switch token.Atom128 {
			case Atom128_DocumentNumber:
				// Validate document format
			case Atom128_PaymentMethod:
				// Validate payment method
			}
		}
	}
	return nil
}

// ============================================================================
// BENCHMARKS AND TESTING
// ============================================================================

func generateJSONWithAtoms() []byte {
	// Generate JSON that benefits from atoms
	obj := map[string]interface{}{
		"id":     "12345",
		"name":   "John Doe",
		"email":  "john@example.com",
		"status": "active",
		"customer": map[string]interface{}{
			"documentNumber": "12345678",
			"documentType":   "DNI",
			"paymentMethod":  "credit_card",
		},
		"items": []map[string]interface{}{
			{"id": "1", "name": "Item 1", "price": 10.99},
			{"id": "2", "name": "Item 2", "price": 20.99},
		},
	}

	data, _ := json.Marshal(obj)
	return data
}

// Helper functions
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ============================================================================
// DEMONSTRATION
// ============================================================================

func main() {
	fmt.Println("Superjsonic with Atom64 and Atom128 Support")
	fmt.Println("===========================================")

	// Generate test data
	testData := generateJSONWithAtoms()
	fmt.Printf("Test JSON: %s\n\n", testData)

	// Parse with atom support
	parser := GetParser()
	defer ReturnParser(parser)

	start := time.Now()
	err := parser.Parse(testData)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
		return
	}
	parseTime := time.Since(start)

	// Display tokens
	fmt.Println("Tokens:")
	for i, token := range parser.GetTokens() {
		switch token.Type {
		case TokenKey64:
			fmt.Printf("[%d] KEY64: atom=%016x\n", i, token.Atom64)
		case TokenKey128:
			fmt.Printf("[%d] KEY128: atom={%016x,%016x}\n", i, token.Atom128.Lo, token.Atom128.Hi)
		case TokenString:
			fmt.Printf("[%d] STRING: %s\n", i, parser.GetTokenValue(token))
		case TokenNumber:
			fmt.Printf("[%d] NUMBER: %s\n", i, parser.GetTokenValue(token))
		default:
			fmt.Printf("[%d] %v\n", i, token.Type)
		}
	}

	fmt.Printf("\nParse time: %v\n", parseTime)
	fmt.Printf("Token count: %d\n", len(parser.GetTokens()))

	// Demonstrate atom matching
	fmt.Println("\nAtom Matching Demo:")
	for _, token := range parser.GetTokens() {
		if token.IsAtom64() {
			switch token.Atom64 {
			case Atom64_Id:
				fmt.Println("Found 'id' field!")
			case Atom64_Email:
				fmt.Println("Found 'email' field!")
			case Atom64_Status:
				fmt.Println("Found 'status' field!")
			}
		}
	}
}
