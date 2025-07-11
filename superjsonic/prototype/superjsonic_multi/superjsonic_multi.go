package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"
)

// ============================================================================
// CONFIGURATION FLAGS
// ============================================================================

type Config struct {
	EnableAtoms      bool
	EnableSIMD       bool
	EnableStringView bool
	EnableFastCompare bool
	EnableArrayOpt   bool
	EnablePooling    bool
	Enable16ByteSIMD bool
	EnableAll        bool
	DisableAll       bool
}

var config Config

func init() {
	flag.BoolVar(&config.EnableAtoms, "atoms", true, "Enable Atom64/Atom128 optimization")
	flag.BoolVar(&config.EnableSIMD, "simd", true, "Enable SIMD operations")
	flag.BoolVar(&config.EnableStringView, "stringview", false, "Enable zero-copy string views")
	flag.BoolVar(&config.EnableFastCompare, "fastcompare", false, "Enable fast token comparison")
	flag.BoolVar(&config.EnableArrayOpt, "array", true, "Enable array detection optimization")
	flag.BoolVar(&config.EnablePooling, "pooling", true, "Enable parser pooling")
	flag.BoolVar(&config.Enable16ByteSIMD, "simd16", false, "Enable 16-byte SIMD chunks")
	flag.BoolVar(&config.EnableAll, "all", false, "Enable all optimizations")
	flag.BoolVar(&config.DisableAll, "none", false, "Disable all optimizations")
}

func applyConfigOverrides() {
	if config.EnableAll {
		config.EnableAtoms = true
		config.EnableSIMD = true
		config.EnableStringView = true
		config.EnableFastCompare = true
		config.EnableArrayOpt = true
		config.EnablePooling = true
		config.Enable16ByteSIMD = true
	} else if config.DisableAll {
		config.EnableAtoms = false
		config.EnableSIMD = false
		config.EnableStringView = false
		config.EnableFastCompare = false
		config.EnableArrayOpt = false
		config.EnablePooling = false
		config.Enable16ByteSIMD = false
	}
}

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
	Atom64_Id     Atom64 = 0x6469           // "id"
	Atom64_Key    Atom64 = 0x79656b         // "key"
	Atom64_Name   Atom64 = 0x656d616e       // "name"
	Atom64_Type   Atom64 = 0x65707974       // "type"
	Atom64_Data   Atom64 = 0x61746164       // "data"
	Atom64_Code   Atom64 = 0x65646f63       // "code"
	Atom64_Text   Atom64 = 0x74786574       // "text"
	Atom64_Value  Atom64 = 0x65756c6176     // "value"
	
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
	end := len(data)
	if end > 8 {
		end = 8
	}
	for i := 0; i < end; i++ {
		atom.Lo |= uint64(data[i]) << (i * 8)
	}
	// Fill high bytes (starting from byte 8)
	for i := 8; i < len(data); i++ {
		atom.Hi |= uint64(data[i]) << ((i-8) * 8)
	}
	return atom
}

func makeAtom128Static(s string) Atom128 {
	return makeAtom128([]byte(s))
}

// Helper to convert atom back to string for debugging
func (a Atom64) String() string {
	var buf [8]byte
	for i := 0; i < 8; i++ {
		b := byte(a >> (i * 8))
		if b == 0 {
			return string(buf[:i])
		}
		buf[i] = b
	}
	return string(buf[:])
}

func (a Atom128) String() string {
	var buf [16]byte
	// Extract from Lo
	for i := 0; i < 8; i++ {
		b := byte(a.Lo >> (i * 8))
		if b == 0 {
			return string(buf[:i])
		}
		buf[i] = b
	}
	// Extract from Hi
	for i := 0; i < 8; i++ {
		b := byte(a.Hi >> (i * 8))
		if b == 0 {
			return string(buf[:i+8])
		}
		buf[i+8] = b
	}
	return string(buf[:])
}

// ============================================================================
// STRING VIEW FOR ZERO-COPY ACCESS
// ============================================================================

type StringView struct {
	ptr unsafe.Pointer
	len int
}

func (sv StringView) String() string {
	if sv.len == 0 {
		return ""
	}
	return *(*string)(unsafe.Pointer(&sv))
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
	TokenKey64   // Key stored as Atom64
	TokenKey128  // Key stored as Atom128
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

func (t Token) IsKey() bool {
	return t.Type == TokenKey64 || t.Type == TokenKey128 || t.Type == TokenString
}

// ============================================================================
// SIMD OPERATIONS - NO CONFIG CHECKS IN HOT PATH!
// ============================================================================

// SIMD operations interface
type SimdOps interface {
	HasByte(v uint64, b byte) bool
	IsWhitespace(v uint64) uint64
}

// Real SIMD implementation
type RealSimdOps struct{}

func (RealSimdOps) HasByte(v uint64, b byte) bool {
	n := uint64(b)
	hasZero := func(v uint64) bool {
		return (v-0x0101010101010101)&^v&0x8080808080808080 != 0
	}
	return hasZero(v ^ (n * 0x0101010101010101))
}

func (RealSimdOps) IsWhitespace(v uint64) uint64 {
	spaces := v ^ 0x2020202020202020
	tabs := v ^ 0x0909090909090909
	lfs := v ^ 0x0A0A0A0A0A0A0A0A
	crs := v ^ 0x0D0D0D0D0D0D0D0D

	hasZero := func(v uint64) uint64 {
		return (v - 0x0101010101010101) & ^v & 0x8080808080808080
	}

	return hasZero(spaces) | hasZero(tabs) | hasZero(lfs) | hasZero(crs)
}

// No-SIMD fallback implementation
type NoSimdOps struct{}

func (NoSimdOps) HasByte(v uint64, b byte) bool {
	bytes := (*[8]byte)(unsafe.Pointer(&v))
	for i := 0; i < 8; i++ {
		if bytes[i] == b {
			return true
		}
	}
	return false
}

func (NoSimdOps) IsWhitespace(v uint64) uint64 {
	bytes := (*[8]byte)(unsafe.Pointer(&v))
	var result uint64
	for i := 0; i < 8; i++ {
		if isWhitespace(bytes[i]) {
			result |= 0x80 << (i * 8)
		}
	}
	return result
}

// ============================================================================
// PARSE STATE MANAGEMENT
// ============================================================================

type ParseContext int

const (
	ContextValue ParseContext = iota
	ContextObjectStart
	ContextObjectKey
	ContextObjectColon
	ContextObjectValue
	ContextObjectComma
	ContextArrayStart
	ContextArrayValue
	ContextArrayComma
)

// ============================================================================
// ENHANCED PARSER WITH CONFIGURABLE OPTIMIZATIONS
// ============================================================================

type ArrayInfo struct {
	isLargeArray      bool
	tokensPerElement  int
	estimatedElements int
}

// Parser operations interface - set once at creation
type ParserOps struct {
	parseString      func(*FastJSONParser) error
	skipWhitespace   func(*FastJSONParser)
	parseLiteral     func(*FastJSONParser) error
	processKey       func(*FastJSONParser, int, int) 
}

type FastJSONParser struct {
	data        []byte
	tokens      []Token
	pos         int
	
	// Operations set once at parser creation
	simd        SimdOps
	ops         ParserOps
	
	// Array optimization
	arrayInfo   ArrayInfo
	
	// Context tracking
	contextStack []ParseContext
	currentContext ParseContext
	
	// Atom statistics
	atomStats   AtomStats
}

type AtomStats struct {
	totalKeys    int
	atom64Keys   int
	atom128Keys  int
	regularKeys  int
}

// Parser pool
var parserPool = sync.Pool{
	New: func() interface{} {
		return createParser()
	},
}

// Create parser with configuration applied ONCE
func createParser() *FastJSONParser {
	p := &FastJSONParser{
		tokens:       make([]Token, 0, 1024),
		contextStack: make([]ParseContext, 0, 32),
	}
	
	// Set SIMD operations based on config
	if config.EnableSIMD {
		p.simd = RealSimdOps{}
	} else {
		p.simd = NoSimdOps{}
	}
	
	// Set parser operations based on config
	p.ops = ParserOps{}
	
	// String parsing
	if config.EnableSIMD {
		if config.Enable16ByteSIMD {
			p.ops.parseString = (*FastJSONParser).parseStringSIMD16
		} else {
			p.ops.parseString = (*FastJSONParser).parseStringSIMD
		}
	} else {
		p.ops.parseString = (*FastJSONParser).parseStringBasic
	}
	
	// Whitespace skipping
	if config.EnableSIMD {
		p.ops.skipWhitespace = (*FastJSONParser).skipWhitespaceSIMD
	} else {
		p.ops.skipWhitespace = (*FastJSONParser).skipWhitespaceBasic
	}
	
	// Literal parsing
	if config.EnableSIMD {
		p.ops.parseLiteral = (*FastJSONParser).parseLiteralFast
	} else {
		p.ops.parseLiteral = (*FastJSONParser).parseLiteralBasic
	}
	
	// Key processing
	if config.EnableAtoms {
		p.ops.processKey = (*FastJSONParser).processKeyAtom
	} else {
		p.ops.processKey = (*FastJSONParser).processKeyString
	}
	
	return p
}

func GetParser() *FastJSONParser {
	if config.EnablePooling {
		return parserPool.Get().(*FastJSONParser)
	}
	return createParser()
}

func ReturnParser(p *FastJSONParser) {
	if config.EnablePooling {
		p.data = nil
		p.tokens = p.tokens[:0]
		p.pos = 0
		p.contextStack = p.contextStack[:0]
		p.currentContext = ContextValue
		p.atomStats = AtomStats{}
		parserPool.Put(p)
	}
}

// Main parse function
func (p *FastJSONParser) Parse(jsonData []byte) error {
	p.data = jsonData
	p.tokens = p.tokens[:0]
	p.pos = 0
	p.contextStack = p.contextStack[:0]
	p.currentContext = ContextValue
	
	// Array detection
	if config.EnableArrayOpt {
		p.arrayInfo = p.detectLargeArray(jsonData)
		if p.arrayInfo.isLargeArray && cap(p.tokens) < p.arrayInfo.estimatedElements*p.arrayInfo.tokensPerElement {
			p.tokens = make([]Token, 0, p.arrayInfo.estimatedElements*p.arrayInfo.tokensPerElement)
		}
	}
	
	return p.parseValue()
}

func (p *FastJSONParser) pushContext(ctx ParseContext) {
	p.contextStack = append(p.contextStack, p.currentContext)
	p.currentContext = ctx
}

func (p *FastJSONParser) popContext() {
	if len(p.contextStack) > 0 {
		p.currentContext = p.contextStack[len(p.contextStack)-1]
		p.contextStack = p.contextStack[:len(p.contextStack)-1]
	} else {
		p.currentContext = ContextValue
	}
}

func (p *FastJSONParser) parseValue() error {
	p.ops.skipWhitespace(p)
	if p.pos >= len(p.data) {
		return fmt.Errorf("unexpected end of JSON")
	}
	
	switch p.data[p.pos] {
	case '{':
		return p.parseObject()
	case '[':
		return p.parseArray()
	case '"':
		return p.ops.parseString(p)
	case 't', 'f', 'n':
		return p.ops.parseLiteral(p)
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return p.parseNumber()
	default:
		return fmt.Errorf("unexpected character '%c' at position %d", p.data[p.pos], p.pos)
	}
}

func (p *FastJSONParser) parseObject() error {
	p.addToken(TokenObjectStart, p.pos, 1, 0, Atom128{})
	p.pos++ // skip '{'
	p.pushContext(ContextObjectStart)
	
	first := true
	for {
		p.ops.skipWhitespace(p)
		if p.pos >= len(p.data) {
			return fmt.Errorf("unterminated object")
		}
		
		// Empty object or end of object
		if p.data[p.pos] == '}' {
			p.addToken(TokenObjectEnd, p.pos, 1, 0, Atom128{})
			p.pos++
			p.popContext()
			return nil
		}
		
		// Need comma between elements (except first)
		if !first {
			if p.data[p.pos] != ',' {
				return fmt.Errorf("expected ',' between object members at position %d", p.pos)
			}
			p.addToken(TokenComma, p.pos, 1, 0, Atom128{})
			p.pos++
			p.ops.skipWhitespace(p)
		}
		first = false
		
		// Parse key
		p.currentContext = ContextObjectKey
		if p.data[p.pos] != '"' {
			return fmt.Errorf("expected '\"' for object key at position %d", p.pos)
		}
		if err := p.ops.parseString(p); err != nil {
			return err
		}
		
		// Parse colon
		p.ops.skipWhitespace(p)
		if p.pos >= len(p.data) || p.data[p.pos] != ':' {
			return fmt.Errorf("expected ':' after object key at position %d", p.pos)
		}
		p.addToken(TokenColon, p.pos, 1, 0, Atom128{})
		p.pos++
		
		// Parse value
		p.currentContext = ContextObjectValue
		if err := p.parseValue(); err != nil {
			return err
		}
		p.currentContext = ContextObjectComma
	}
}

func (p *FastJSONParser) parseArray() error {
	p.addToken(TokenArrayStart, p.pos, 1, 0, Atom128{})
	p.pos++ // skip '['
	p.pushContext(ContextArrayStart)
	
	first := true
	for {
		p.ops.skipWhitespace(p)
		if p.pos >= len(p.data) {
			return fmt.Errorf("unterminated array")
		}
		
		// Empty array or end of array
		if p.data[p.pos] == ']' {
			p.addToken(TokenArrayEnd, p.pos, 1, 0, Atom128{})
			p.pos++
			p.popContext()
			return nil
		}
		
		// Need comma between elements (except first)
		if !first {
			if p.data[p.pos] != ',' {
				return fmt.Errorf("expected ',' between array elements at position %d", p.pos)
			}
			p.addToken(TokenComma, p.pos, 1, 0, Atom128{})
			p.pos++
			p.ops.skipWhitespace(p)
		}
		first = false
		
		// Parse value
		p.currentContext = ContextArrayValue
		if err := p.parseValue(); err != nil {
			return err
		}
		p.currentContext = ContextArrayComma
	}
}

// String parsing implementations
func (p *FastJSONParser) parseStringSIMD16() error {
	start := p.pos
	p.pos++ // skip opening quote
	stringStart := p.pos
	
	// 16-byte SIMD scanning
	for p.pos+16 <= len(p.data) {
		v1 := *(*uint64)(unsafe.Pointer(&p.data[p.pos]))
		v2 := *(*uint64)(unsafe.Pointer(&p.data[p.pos+8]))
		
		if p.simd.HasByte(v1, '"') || p.simd.HasByte(v1, '\\') ||
		   p.simd.HasByte(v2, '"') || p.simd.HasByte(v2, '\\') {
			// Found delimiter, check which byte
			for j := 0; j < 16; j++ {
				if p.data[p.pos+j] == '\\' {
					p.pos += j + 2
					goto continueString16
				} else if p.data[p.pos+j] == '"' {
					p.pos += j
					goto stringEnd
				}
			}
		}
		p.pos += 16
	continueString16:
	}
	
	// Fall through to 8-byte scanning for remainder
	return p.parseStringSIMD8(start, stringStart)
	
stringEnd:
	p.finishString(start, stringStart)
	return nil
}

func (p *FastJSONParser) parseStringSIMD() error {
	start := p.pos
	p.pos++ // skip opening quote
	stringStart := p.pos
	
	return p.parseStringSIMD8(start, stringStart)
}

func (p *FastJSONParser) parseStringSIMD8(start, stringStart int) error {
	// Regular 8-byte SIMD scanning
	for p.pos+8 <= len(p.data) {
		v := *(*uint64)(unsafe.Pointer(&p.data[p.pos]))
		
		if p.simd.HasByte(v, '"') || p.simd.HasByte(v, '\\') {
			for j := 0; j < 8; j++ {
				if p.data[p.pos+j] == '\\' {
					p.pos += j + 2
					goto continueString8
				} else if p.data[p.pos+j] == '"' {
					p.pos += j
					goto stringEnd
				}
			}
		}
		p.pos += 8
	continueString8:
	}
	
	// Handle remaining bytes
	for p.pos < len(p.data) && p.data[p.pos] != '"' {
		if p.data[p.pos] == '\\' {
			p.pos++
			if p.pos >= len(p.data) {
				return fmt.Errorf("unterminated string escape")
			}
		}
		p.pos++
	}
	
stringEnd:
	if p.pos >= len(p.data) {
		return fmt.Errorf("unterminated string")
	}
	
	p.finishString(start, stringStart)
	return nil
}

func (p *FastJSONParser) parseStringBasic() error {
	start := p.pos
	p.pos++ // skip opening quote
	stringStart := p.pos
	
	// Basic byte-by-byte scanning
	for p.pos < len(p.data) && p.data[p.pos] != '"' {
		if p.data[p.pos] == '\\' {
			p.pos++
			if p.pos >= len(p.data) {
				return fmt.Errorf("unterminated string escape")
			}
		}
		p.pos++
	}
	
	if p.pos >= len(p.data) {
		return fmt.Errorf("unterminated string")
	}
	
	p.finishString(start, stringStart)
	return nil
}

func (p *FastJSONParser) finishString(start, stringStart int) {
	if p.currentContext == ContextObjectKey {
		keyLen := p.pos - stringStart
		p.ops.processKey(p, stringStart, keyLen)
	} else {
		// Regular string token for values
		p.addToken(TokenString, start, p.pos-start+1, 0, Atom128{})
	}
	p.pos++ // skip closing quote
}

// Key processing implementations
func (p *FastJSONParser) processKeyAtom(stringStart, keyLen int) {
	p.atomStats.totalKeys++
	
	if keyLen <= 8 {
		atom := makeAtom64(p.data[stringStart:stringStart+keyLen])
		p.addToken(TokenKey64, 0, 0, atom, Atom128{})
		p.atomStats.atom64Keys++
	} else if keyLen <= 16 {
		atom := makeAtom128(p.data[stringStart:stringStart+keyLen])
		p.addToken(TokenKey128, 0, 0, 0, atom)
		p.atomStats.atom128Keys++
	} else {
		// Regular string for long keys
		p.addToken(TokenString, stringStart-1, keyLen+2, 0, Atom128{})
		p.atomStats.regularKeys++
	}
}

func (p *FastJSONParser) processKeyString(stringStart, keyLen int) {
	p.atomStats.totalKeys++
	p.atomStats.regularKeys++
	p.addToken(TokenString, stringStart-1, keyLen+2, 0, Atom128{})
}

// Whitespace skipping implementations
func (p *FastJSONParser) skipWhitespaceSIMD() {
	// SIMD-optimized whitespace skipping
	for p.pos+8 <= len(p.data) {
		v := *(*uint64)(unsafe.Pointer(&p.data[p.pos]))
		wsResult := p.simd.IsWhitespace(v)
		if wsResult == 0 {
			return
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

func (p *FastJSONParser) skipWhitespaceBasic() {
	for p.pos < len(p.data) && isWhitespace(p.data[p.pos]) {
		p.pos++
	}
}

// Literal parsing implementations
func (p *FastJSONParser) parseLiteralFast() error {
	switch p.data[p.pos] {
	case 't':
		if p.pos+4 <= len(p.data) && *(*uint32)(unsafe.Pointer(&p.data[p.pos])) == 0x65757274 {
			p.addToken(TokenTrue, p.pos, 4, 0, Atom128{})
			p.pos += 4
			return nil
		}
	case 'f':
		if p.pos+5 <= len(p.data) && *(*uint32)(unsafe.Pointer(&p.data[p.pos])) == 0x736c6166 && p.data[p.pos+4] == 'e' {
			p.addToken(TokenFalse, p.pos, 5, 0, Atom128{})
			p.pos += 5
			return nil
		}
	case 'n':
		if p.pos+4 <= len(p.data) && *(*uint32)(unsafe.Pointer(&p.data[p.pos])) == 0x6c6c756e {
			p.addToken(TokenNull, p.pos, 4, 0, Atom128{})
			p.pos += 4
			return nil
		}
	}
	return fmt.Errorf("invalid literal at position %d", p.pos)
}

func (p *FastJSONParser) parseLiteralBasic() error {
	start := p.pos
	
	switch p.data[p.pos] {
	case 't':
		if p.pos+4 <= len(p.data) && string(p.data[p.pos:p.pos+4]) == "true" {
			p.addToken(TokenTrue, start, 4, 0, Atom128{})
			p.pos += 4
			return nil
		}
	case 'f':
		if p.pos+5 <= len(p.data) && string(p.data[p.pos:p.pos+5]) == "false" {
			p.addToken(TokenFalse, start, 5, 0, Atom128{})
			p.pos += 5
			return nil
		}
	case 'n':
		if p.pos+4 <= len(p.data) && string(p.data[p.pos:p.pos+4]) == "null" {
			p.addToken(TokenNull, start, 4, 0, Atom128{})
			p.pos += 4
			return nil
		}
	}
	return fmt.Errorf("invalid literal at position %d", p.pos)
}

func (p *FastJSONParser) parseNumber() error {
	start := p.pos
	
	// Optional minus
	if p.data[p.pos] == '-' {
		p.pos++
	}
	
	// Must have at least one digit
	if p.pos >= len(p.data) || !isDigit(p.data[p.pos]) {
		return fmt.Errorf("invalid number at position %d", start)
	}
	
	// Integer part
	if p.data[p.pos] == '0' {
		p.pos++
	} else {
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			p.pos++
		}
	}
	
	// Fractional part
	if p.pos < len(p.data) && p.data[p.pos] == '.' {
		p.pos++
		if p.pos >= len(p.data) || !isDigit(p.data[p.pos]) {
			return fmt.Errorf("invalid number: decimal point must be followed by digits")
		}
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			p.pos++
		}
	}
	
	// Exponent part
	if p.pos < len(p.data) && (p.data[p.pos] == 'e' || p.data[p.pos] == 'E') {
		p.pos++
		if p.pos < len(p.data) && (p.data[p.pos] == '+' || p.data[p.pos] == '-') {
			p.pos++
		}
		if p.pos >= len(p.data) || !isDigit(p.data[p.pos]) {
			return fmt.Errorf("invalid number: exponent must have digits")
		}
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			p.pos++
		}
	}
	
	p.addToken(TokenNumber, start, p.pos-start, 0, Atom128{})
	return nil
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

// Array detection
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
		`"transactions":[`,
		`"orders":[`,
		`"customers":[`,
	}
	
	dataStr := string(data[:min(1000, len(data))])
	for _, pattern := range patterns {
		if idx := strings.Index(dataStr, pattern); idx >= 0 {
			// Estimate array size
			arrayStart := idx + len(pattern)
			if arrayStart < len(data) {
				elementSample := p.sampleArrayElement(data[arrayStart:])
				if elementSample > 10 {
					estimatedElements := (len(data) - arrayStart) / elementSample
					tokensPerElement := 20 // rough estimate
					return ArrayInfo{
						isLargeArray:      true,
						tokensPerElement:  tokensPerElement,
						estimatedElements: estimatedElements,
					}
				}
			}
		}
	}
	
	// Check if entire data is array
	if len(data) > 0 && data[0] == '[' {
		elementSample := p.sampleArrayElement(data[1:])
		if elementSample > 10 {
			estimatedElements := len(data) / elementSample
			return ArrayInfo{
				isLargeArray:      true,
				tokensPerElement:  20,
				estimatedElements: estimatedElements,
			}
		}
	}
	
	return ArrayInfo{}
}

func (p *FastJSONParser) sampleArrayElement(data []byte) int {
	depth := 0
	inString := false
	escape := false
	
	for i := 0; i < len(data) && i < 10000; i++ {
		b := data[i]
		
		if !inString {
			switch b {
			case '"':
				if !escape {
					inString = true
				}
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth == 0 {
					return i + 1
				}
			case ',':
				if depth == 0 {
					return i
				}
			}
		} else {
			if b == '"' && !escape {
				inString = false
			}
		}
		
		escape = !escape && b == '\\'
	}
	return 0
}

// Token access methods
func (p *FastJSONParser) GetTokens() []Token {
	return p.tokens
}

func (p *FastJSONParser) GetTokenValue(token Token) string {
	if token.IsAtom64() {
		return token.Atom64.String()
	}
	if token.IsAtom128() {
		return token.Atom128.String()
	}
	if token.Offset < uint32(len(p.data)) && token.Offset+token.Length <= uint32(len(p.data)) {
		return string(p.data[token.Offset : token.Offset+token.Length])
	}
	return ""
}

// Zero-copy string view access
func (p *FastJSONParser) GetTokenStringView(token Token) StringView {
	if !config.EnableStringView || token.IsAtom64() || token.IsAtom128() {
		return StringView{}
	}
	return StringView{
		ptr: unsafe.Pointer(&p.data[token.Offset]),
		len: int(token.Length),
	}
}

// Fast token comparison
func (p *FastJSONParser) CompareToken(token Token, target string) bool {
	if !config.EnableFastCompare {
		// Fallback to regular string comparison
		return p.GetTokenValue(token) == target
	}
	
	if token.IsAtom64() {
		targetAtom := makeAtom64([]byte(target))
		return token.Atom64 == targetAtom
	}
	if token.IsAtom128() {
		targetAtom := makeAtom128([]byte(target))
		return token.Atom128.Lo == targetAtom.Lo && token.Atom128.Hi == targetAtom.Hi
	}
	
	// For regular strings, use fast comparison
	if int(token.Length) != len(target) {
		return false
	}
	
	tokenData := p.data[token.Offset : token.Offset+token.Length]
	targetBytes := []byte(target)
	
	// SIMD comparison if enabled
	if config.EnableSIMD {
		i := 0
		for i+8 <= len(target) {
			if *(*uint64)(unsafe.Pointer(&tokenData[i])) != *(*uint64)(unsafe.Pointer(&targetBytes[i])) {
				return false
			}
			i += 8
		}
		
		// Compare remaining bytes
		for ; i < len(target); i++ {
			if tokenData[i] != targetBytes[i] {
				return false
			}
		}
		
		return true
	}
	
	// Non-SIMD comparison
	for i := 0; i < len(target); i++ {
		if tokenData[i] != targetBytes[i] {
			return false
		}
	}
	return true
}

func (p *FastJSONParser) GetStats() AtomStats {
	return p.atomStats
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
// BENCHMARKING AND FILE TESTING
// ============================================================================

type BenchmarkResult struct {
	Filename      string
	FileSize      int64
	ParseTime     time.Duration
	TokenCount    int
	AtomStats     AtomStats
	Throughput    float64 // MB/s
	AtomsPercent  float64
	Config        string
}

func benchmarkFile(filename string) (BenchmarkResult, error) {
	// Read file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return BenchmarkResult{}, err
	}
	
	// Get file info
	info, err := os.Stat(filename)
	if err != nil {
		return BenchmarkResult{}, err
	}
	
	// Parse with timing
	parser := GetParser()
	defer ReturnParser(parser)
	
	start := time.Now()
	err = parser.Parse(data)
	parseTime := time.Since(start)
	
	if err != nil {
		return BenchmarkResult{}, fmt.Errorf("parse error: %v", err)
	}
	
	// Calculate stats
	stats := parser.GetStats()
	throughput := float64(len(data)) / parseTime.Seconds() / 1024 / 1024
	atomsPercent := 0.0
	if stats.totalKeys > 0 {
		atomsPercent = float64(stats.atom64Keys+stats.atom128Keys) / float64(stats.totalKeys) * 100
	}
	
	return BenchmarkResult{
		Filename:     filename,
		FileSize:     info.Size(),
		ParseTime:    parseTime,
		TokenCount:   len(parser.GetTokens()),
		AtomStats:    stats,
		Throughput:   throughput,
		AtomsPercent: atomsPercent,
		Config:       getConfigString(),
	}, nil
}

func benchmarkComparison(filename string) error {
	// Read file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	
	// Benchmark Superjsonic with current config
	parser := GetParser()
	start := time.Now()
	err = parser.Parse(data)
	superjsonicTime := time.Since(start)
	superjsonicTokens := len(parser.GetTokens())
	ReturnParser(parser)
	
	if err != nil {
		return fmt.Errorf("superjsonic parse error: %v", err)
	}
	
	// Benchmark standard library
	var jsonData interface{}
	start = time.Now()
	err = json.Unmarshal(data, &jsonData)
	stdTime := time.Since(start)
	
	if err != nil {
		return fmt.Errorf("stdlib parse error: %v", err)
	}
	
	// Print comparison
	fmt.Printf("\n%s Comparison:\n", filepath.Base(filename))
	fmt.Printf("  Superjsonic: %v (%d tokens, %.2f MB/s)\n", 
		superjsonicTime, superjsonicTokens,
		float64(len(data))/superjsonicTime.Seconds()/1024/1024)
	fmt.Printf("  Standard lib: %v (%.2f MB/s)\n", 
		stdTime,
		float64(len(data))/stdTime.Seconds()/1024/1024)
	fmt.Printf("  Speedup: %.2fx\n", float64(stdTime)/float64(superjsonicTime))
	
	return nil
}

func getConfigString() string {
	var opts []string
	if config.EnableAtoms {
		opts = append(opts, "atoms")
	}
	if config.EnableSIMD {
		opts = append(opts, "simd")
	}
	if config.EnableStringView {
		opts = append(opts, "stringview")
	}
	if config.EnableFastCompare {
		opts = append(opts, "fastcompare")
	}
	if config.EnableArrayOpt {
		opts = append(opts, "array")
	}
	if config.EnablePooling {
		opts = append(opts, "pooling")
	}
	if config.Enable16ByteSIMD {
		opts = append(opts, "simd16")
	}
	
	if len(opts) == 0 {
		return "none"
	}
	return strings.Join(opts, "+")
}

// ============================================================================
// MAIN FUNCTION
// ============================================================================

func main() {
	flag.Parse()
	applyConfigOverrides()
	
	fmt.Println("Superjsonic with Optimized Configuration Checks")
	fmt.Println("===============================================")
	fmt.Printf("Configuration: %s\n\n", getConfigString())
	
	// Test files
	testFiles := []string{
		"file1_small.json",
		"file2_medium.json",
		"file3_large.json",
		"file4_xlarge.json",
		"file5_xxlarge.json",
	}
	
	// Benchmark each file
	results := make([]BenchmarkResult, 0, len(testFiles))
	for _, filename := range testFiles {
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			fmt.Printf("Skipping %s (not found)\n", filename)
			continue
		}
		
		result, err := benchmarkFile(filename)
		if err != nil {
			fmt.Printf("Error benchmarking %s: %v\n", filename, err)
			continue
		}
		results = append(results, result)
	}
	
	// Print results table
	fmt.Println("\nResults Summary:")
	fmt.Println("File            Size      Parse Time   Tokens    Throughput   Atoms%   Atom64   Atom128  Regular")
	fmt.Println("----            ----      ----------   ------    ----------   ------   ------   -------  -------")
	
	for _, r := range results {
		atomPercent := "-"
		if config.EnableAtoms {
			atomPercent = fmt.Sprintf("%.1f", r.AtomsPercent)
		}
		
		fmt.Printf("%-15s %-9s %-12v %-9d %-12.2f %-8s %-8d %-8d %-8d\n",
			filepath.Base(r.Filename),
			formatSize(r.FileSize),
			r.ParseTime,
			r.TokenCount,
			r.Throughput,
			atomPercent,
			r.AtomStats.atom64Keys,
			r.AtomStats.atom128Keys,
			r.AtomStats.regularKeys,
		)
	}
	
	// Run comparison benchmarks
	if len(results) > 0 {
		fmt.Println("\n\nComparison with Standard Library:")
		fmt.Println("---------------------------------")
		for _, filename := range testFiles[:3] { // Just first 3 files for comparison
			if _, err := os.Stat(filename); os.IsNotExist(err) {
				continue
			}
			benchmarkComparison(filename)
		}
	}
	
	// Show configuration impact
	if len(results) > 0 {
		fmt.Printf("\n\nOptimizations Enabled: %s\n", getConfigString())
		fmt.Println("\nUsage examples:")
		fmt.Println("  Disable all:     go run main.go -none")
		fmt.Println("  Only SIMD:       go run main.go -none -simd")
		fmt.Println("  Only atoms:      go run main.go -none -atoms")
		fmt.Println("  No atoms:        go run main.go -atoms=false")
		fmt.Println("  Everything:      go run main.go -all")
		fmt.Println("  With stringview: go run main.go -stringview")
	}
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}