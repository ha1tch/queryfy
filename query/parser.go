package query

import (
	"fmt"
	"strconv"
)

// Parser parses query strings into AST.
type Parser struct {
	lexer   *Lexer
	current Token
	peek    Token
}

// NewParser creates a new parser for the input string.
func NewParser(input string) *Parser {
	lexer := NewLexer(input)
	p := &Parser{
		lexer: lexer,
	}
	// Prime the parser with two tokens
	p.advance()
	p.advance()
	return p
}

// Parse parses the input and returns a Query AST.
func (p *Parser) Parse() (*Query, error) {
	if p.current.Type == TokenEOF {
		return nil, fmt.Errorf("empty query")
	}
	
	root, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	
	if p.current.Type != TokenEOF {
		return nil, fmt.Errorf("unexpected token at position %d: %s", p.current.Pos, p.current.Value)
	}
	
	return &Query{
		Root: &RootNode{Child: root},
	}, nil
}

// parseExpression parses a query expression.
func (p *Parser) parseExpression() (Node, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	
	for p.current.Type == TokenDot {
		p.advance() // consume dot
		
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		
		left = &DotNode{
			Left:  left,
			Right: right,
		}
	}
	
	return left, nil
}

// parsePrimary parses a primary expression (identifier or identifier[index]).
func (p *Parser) parsePrimary() (Node, error) {
	if p.current.Type != TokenIdentifier {
		return nil, fmt.Errorf("expected identifier at position %d, got %s", 
			p.current.Pos, TokenTypeName(p.current.Type))
	}
	
	// Change from *IdentifierNode to Node interface type
	var node Node = &IdentifierNode{Name: p.current.Value}
	p.advance()
	
	// Check for array index
	for p.current.Type == TokenLeftBracket {
		p.advance() // consume '['
		
		if p.current.Type != TokenNumber {
			return nil, fmt.Errorf("expected number after '[' at position %d", p.current.Pos)
		}
		
		index, err := strconv.Atoi(p.current.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid array index at position %d: %s", 
				p.current.Pos, p.current.Value)
		}
		
		p.advance() // consume number
		
		if p.current.Type != TokenRightBracket {
			return nil, fmt.Errorf("expected ']' at position %d", p.current.Pos)
		}
		
		p.advance() // consume ']'
		
		// Create a composite node for array access
		node = &DotNode{
			Left:  node,
			Right: &IndexNode{Index: index},
		}
	}
	
	return node, nil
}

// advance moves to the next token.
func (p *Parser) advance() {
	p.current = p.peek
	p.peek = p.lexer.Next()
}

// ParseQuery is a convenience function that parses a query string.
func ParseQuery(input string) (*Query, error) {
	parser := NewParser(input)
	return parser.Parse()
}

// SimplifyNode simplifies an AST node for easier execution.
// This converts complex nodes into a linear path.
func SimplifyNode(node Node) []interface{} {
	var path []interface{}
	
	var traverse func(n Node)
	traverse = func(n Node) {
		switch n := n.(type) {
		case *IdentifierNode:
			path = append(path, n.Name)
		case *IndexNode:
			path = append(path, n.Index)
		case *DotNode:
			traverse(n.Left)
			traverse(n.Right)
		case *RootNode:
			if n.Child != nil {
				traverse(n.Child)
			}
		}
	}
	
	traverse(node)
	return path
}

// PathFromQuery converts a query string directly to a path.
// This is a convenience function for simple queries.
func PathFromQuery(queryStr string) ([]interface{}, error) {
	query, err := ParseQuery(queryStr)
	if err != nil {
		return nil, err
	}
	
	if query.Root == nil || query.Root.Child == nil {
		return []interface{}{}, nil
	}
	
	return SimplifyNode(query.Root.Child), nil
}