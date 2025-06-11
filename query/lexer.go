package query

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType represents the type of a token.
type TokenType int

const (
	// TokenEOF represents end of input
	TokenEOF TokenType = iota
	// TokenIdentifier represents a field name
	TokenIdentifier
	// TokenDot represents a dot separator
	TokenDot
	// TokenLeftBracket represents '['
	TokenLeftBracket
	// TokenRightBracket represents ']'
	TokenRightBracket
	// TokenNumber represents a numeric value
	TokenNumber
	// TokenError represents a lexing error
	TokenError
)

// Token represents a lexical token.
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// String returns the string representation of a token.
func (t Token) String() string {
	switch t.Type {
	case TokenEOF:
		return "EOF"
	case TokenError:
		return fmt.Sprintf("ERROR:%s", t.Value)
	default:
		return fmt.Sprintf("%v:%s", t.Type, t.Value)
	}
}

// Lexer tokenizes query strings.
type Lexer struct {
	input string
	pos   int
	width int
}

// NewLexer creates a new lexer for the input string.
func NewLexer(input string) *Lexer {
	return &Lexer{
		input: input,
		pos:   0,
	}
}

// Next returns the next token.
func (l *Lexer) Next() Token {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Pos: l.pos}
	}

	ch := l.input[l.pos]

	switch ch {
	case '.':
		l.pos++
		return Token{Type: TokenDot, Value: ".", Pos: l.pos - 1}
	case '[':
		l.pos++
		return Token{Type: TokenLeftBracket, Value: "[", Pos: l.pos - 1}
	case ']':
		l.pos++
		return Token{Type: TokenRightBracket, Value: "]", Pos: l.pos - 1}
	default:
		if unicode.IsDigit(rune(ch)) {
			return l.lexNumber()
		}
		if unicode.IsLetter(rune(ch)) || ch == '_' {
			return l.lexIdentifier()
		}
		return Token{
			Type:  TokenError,
			Value: fmt.Sprintf("unexpected character: %c", ch),
			Pos:   l.pos,
		}
	}
}

// Peek returns the next token without consuming it.
func (l *Lexer) Peek() Token {
	savedPos := l.pos
	token := l.Next()
	l.pos = savedPos
	return token
}

// lexIdentifier lexes an identifier.
func (l *Lexer) lexIdentifier() Token {
	start := l.pos

	for l.pos < len(l.input) {
		ch := rune(l.input[l.pos])
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
			break
		}
		l.pos++
	}

	return Token{
		Type:  TokenIdentifier,
		Value: l.input[start:l.pos],
		Pos:   start,
	}
}

// lexNumber lexes a number.
func (l *Lexer) lexNumber() Token {
	start := l.pos

	for l.pos < len(l.input) && unicode.IsDigit(rune(l.input[l.pos])) {
		l.pos++
	}

	return Token{
		Type:  TokenNumber,
		Value: l.input[start:l.pos],
		Pos:   start,
	}
}

// skipWhitespace skips whitespace characters.
func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
		l.pos++
	}
}

// Tokenize returns all tokens from the input string.
// This is a convenience function for testing.
func Tokenize(input string) []Token {
	lexer := NewLexer(input)
	var tokens []Token

	for {
		token := lexer.Next()
		tokens = append(tokens, token)
		if token.Type == TokenEOF || token.Type == TokenError {
			break
		}
	}

	return tokens
}

// TokenTypeName returns the name of a token type.
func TokenTypeName(t TokenType) string {
	names := []string{
		"EOF",
		"Identifier",
		"Dot",
		"LeftBracket",
		"RightBracket",
		"Number",
		"Error",
	}

	if int(t) < len(names) {
		return names[t]
	}
	return "Unknown"
}

// FormatTokens formats a slice of tokens for debugging.
func FormatTokens(tokens []Token) string {
	var parts []string
	for _, token := range tokens {
		parts = append(parts, token.String())
	}
	return strings.Join(parts, " ")
}
