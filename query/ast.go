package query

// NodeType represents the type of AST node.
type NodeType int

const (
	// NodeIdentifier represents a field name or identifier
	NodeIdentifier NodeType = iota
	// NodeIndex represents an array index access
	NodeIndex
	// NodeDot represents a dot notation access
	NodeDot
	// NodeRoot represents the root of the query
	NodeRoot
)

// Node represents a node in the query AST.
type Node interface {
	Type() NodeType
	String() string
}

// IdentifierNode represents a field name.
type IdentifierNode struct {
	Name string
}

// Type returns the node type.
func (n *IdentifierNode) Type() NodeType {
	return NodeIdentifier
}

// String returns the string representation.
func (n *IdentifierNode) String() string {
	return n.Name
}

// IndexNode represents an array index access.
type IndexNode struct {
	Index int
}

// Type returns the node type.
func (n *IndexNode) Type() NodeType {
	return NodeIndex
}

// String returns the string representation.
func (n *IndexNode) String() string {
	return "[" + string(rune(n.Index+'0')) + "]"
}

// DotNode represents a dot notation access.
type DotNode struct {
	Left  Node
	Right Node
}

// Type returns the node type.
func (n *DotNode) Type() NodeType {
	return NodeDot
}

// String returns the string representation.
func (n *DotNode) String() string {
	return n.Left.String() + "." + n.Right.String()
}

// RootNode represents the root of a query.
type RootNode struct {
	Child Node
}

// Type returns the node type.
func (n *RootNode) Type() NodeType {
	return NodeRoot
}

// String returns the string representation.
func (n *RootNode) String() string {
	if n.Child != nil {
		return n.Child.String()
	}
	return ""
}

// Query represents a parsed query.
type Query struct {
	Root *RootNode
}

// String returns the string representation of the query.
func (q *Query) String() string {
	if q.Root != nil {
		return q.Root.String()
	}
	return ""
}
