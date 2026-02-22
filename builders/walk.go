// walk.go - Schema tree traversal
package builders

import (
	"fmt"

	"github.com/ha1tch/queryfy"
)

// FieldVisitor is called for each node during schema traversal.
// The path argument is the dot-notation path to the current node
// (e.g., "user.address.street" or "items[*].name").
// The schema argument is the schema at that node.
type FieldVisitor func(path string, schema queryfy.Schema) error

// Walk traverses the schema tree depth-first, calling the visitor at
// each node. Object fields are visited with their full dot-notation
// path. Array element schemas are visited with [*] appended.
//
// The root schema itself is visited with an empty path "".
//
// If the visitor returns a non-nil error, traversal stops and Walk
// returns that error.
func Walk(schema queryfy.Schema, visitor FieldVisitor) error {
	return walkNode("", schema, visitor)
}

func walkNode(path string, schema queryfy.Schema, visitor FieldVisitor) error {
	// Visit this node
	if err := visitor(path, schema); err != nil {
		return err
	}

	// Recurse into children based on type
	switch s := schema.(type) {
	case *ObjectSchema:
		return walkObject(path, s, visitor)

	case *ObjectSchemaWithDependencies:
		return walkObject(path, s.ObjectSchema, visitor)

	case *ArraySchema:
		elem := s.ElementSchema()
		if elem != nil {
			childPath := appendPath(path, "[*]")
			if err := walkNode(childPath, elem, visitor); err != nil {
				return err
			}
		}

	case *TransformSchema:
		// TransformSchema wraps an inner schema. We visit the inner
		// schema at the same path since the transform doesn't change
		// the structural position.
		inner := s.InnerSchema()
		if inner != nil {
			if err := walkNode(path, inner, visitor); err != nil {
				return err
			}
		}

	case *AndSchema:
		// Composite: visit each sub-schema
		for i, sub := range s.Schemas() {
			childPath := fmt.Sprintf("%s<and[%d]>", path, i)
			if err := walkNode(childPath, sub, visitor); err != nil {
				return err
			}
		}

	case *OrSchema:
		for i, sub := range s.Schemas() {
			childPath := fmt.Sprintf("%s<or[%d]>", path, i)
			if err := walkNode(childPath, sub, visitor); err != nil {
				return err
			}
		}

	case *NotSchema:
		inner := s.InnerSchema()
		if inner != nil {
			childPath := fmt.Sprintf("%s<not>", path)
			if err := walkNode(childPath, inner, visitor); err != nil {
				return err
			}
		}
	}

	return nil
}

func walkObject(path string, obj *ObjectSchema, visitor FieldVisitor) error {
	for _, name := range obj.FieldNames() {
		field, _ := obj.GetField(name)
		childPath := appendPath(path, name)
		if err := walkNode(childPath, field, visitor); err != nil {
			return err
		}
	}
	return nil
}

// appendPath joins path segments with dots, handling the root case.
func appendPath(base, segment string) string {
	if base == "" {
		return segment
	}
	// Array notation doesn't get a dot prefix
	if len(segment) > 0 && segment[0] == '[' {
		return base + segment
	}
	return base + "." + segment
}
