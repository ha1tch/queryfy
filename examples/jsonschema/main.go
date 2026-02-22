package main

import (
	"encoding/json"
	"fmt"
	"os"

	qf "github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
	"github.com/ha1tch/queryfy/builders/jsonschema"
)

func main() {
	fmt.Println("=== Queryfy JSON Schema Examples ===")

	importExample()
	exportExample()
	roundTripExample()
	migrationExample()
	strictModeExample()
	metadataExample()
}

// importExample shows how to import a JSON Schema document and use it
// for validation with queryfy.
func importExample() {
	fmt.Println("\n1. Importing JSON Schema")
	fmt.Println("------------------------")

	// A JSON Schema document — this could come from a file, an API
	// specification, or a schema registry.
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"email": {
				"type": "string",
				"format": "email",
				"maxLength": 254
			},
			"age": {
				"type": "integer",
				"minimum": 13,
				"maximum": 150
			},
			"role": {
				"type": "string",
				"enum": ["admin", "editor", "viewer"]
			}
		},
		"required": ["email", "role"],
		"additionalProperties": false
	}`)

	// Import the JSON Schema. The second argument controls options;
	// nil uses defaults (non-strict, no metadata storage).
	schema, errs := jsonschema.FromJSON(schemaJSON, nil)
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Printf("  conversion issue: %s\n", e.Error())
		}
		os.Exit(1)
	}

	// The returned schema is a standard queryfy schema — validate as usual.
	valid := map[string]interface{}{
		"email": "alice@example.com",
		"age":   30.0,
		"role":  "admin",
	}
	if err := qf.Validate(valid, schema); err != nil {
		fmt.Printf("  unexpected: %v\n", err)
	} else {
		fmt.Println("  valid user: ok")
	}

	// Missing required field
	incomplete := map[string]interface{}{
		"email": "bob@example.com",
	}
	if err := qf.Validate(incomplete, schema); err != nil {
		fmt.Printf("  missing role: %v\n", err)
	}

	// Bad enum value
	badRole := map[string]interface{}{
		"email": "carol@example.com",
		"role":  "superadmin",
	}
	if err := qf.Validate(badRole, schema); err != nil {
		fmt.Printf("  bad role: %v\n", err)
	}
}

// exportExample shows how to convert a queryfy schema to JSON Schema.
func exportExample() {
	fmt.Println("\n2. Exporting to JSON Schema")
	fmt.Println("---------------------------")

	// Build a schema using queryfy's builder API.
	schema := builders.Object().
		Field("name", builders.String().Required().MinLength(1).MaxLength(100)).
		Field("score", builders.Number().Integer().Min(0).Max(100)).
		Field("tags", builders.Array().Of(builders.String().MaxLength(50)).MaxItems(10).UniqueItems()).
		AllowAdditional(false)

	// Export to JSON Schema with optional metadata.
	data, err := jsonschema.ToJSON(schema, &jsonschema.ExportOptions{
		SchemaURI: "https://json-schema.org/draft/2020-12/schema",
		ID:        "https://example.com/player.schema.json",
	})
	if err != nil {
		fmt.Printf("  export error: %v\n", err)
		return
	}

	fmt.Println("  exported JSON Schema:")
	// Pretty-print
	var pretty json.RawMessage = data
	formatted, _ := json.MarshalIndent(pretty, "  ", "  ")
	fmt.Printf("  %s\n", formatted)
}

// roundTripExample demonstrates import → export → re-import fidelity.
func roundTripExample() {
	fmt.Println("\n3. Round-trip: Import → Export → Re-import")
	fmt.Println("-------------------------------------------")

	original := []byte(`{
		"type": "object",
		"properties": {
			"latitude": {"type": "number", "minimum": -90, "maximum": 90},
			"longitude": {"type": "number", "minimum": -180, "maximum": 180}
		},
		"required": ["latitude", "longitude"]
	}`)

	// Step 1: Import
	schema1, _ := jsonschema.FromJSON(original, nil)

	// Step 2: Export
	exported, _ := jsonschema.ToJSON(schema1, nil)

	// Step 3: Re-import
	schema2, _ := jsonschema.FromJSON(exported, nil)

	// Both schemas validate identically.
	coord := map[string]interface{}{
		"latitude":  51.5074,
		"longitude": -0.1278,
	}
	err1 := qf.Validate(coord, schema1)
	err2 := qf.Validate(coord, schema2)
	fmt.Printf("  original schema:     %v\n", err1 == nil)
	fmt.Printf("  round-tripped schema: %v\n", err2 == nil)

	badCoord := map[string]interface{}{
		"latitude":  999.0,
		"longitude": -0.1278,
	}
	err1 = qf.Validate(badCoord, schema1)
	err2 = qf.Validate(badCoord, schema2)
	fmt.Printf("  original rejects bad:     %v\n", err1 != nil)
	fmt.Printf("  round-tripped rejects bad: %v\n", err2 != nil)
}

// migrationExample shows how to use import + export to migrate between
// a JSON Schema definition and queryfy's builder API.
func migrationExample() {
	fmt.Println("\n4. Migration: JSON Schema → queryfy builders")
	fmt.Println("----------------------------------------------")
	fmt.Println("  If your system already defines contracts in JSON Schema,")
	fmt.Println("  you can import them and validate with queryfy without")
	fmt.Println("  rewriting your schema definitions:")
	fmt.Println()

	// Load from an external source (here, inline for demonstration).
	externalSchema := []byte(`{
		"type": "object",
		"properties": {
			"transaction_id": {"type": "string", "pattern": "^TXN-[0-9]{8}$"},
			"amount": {"type": "number", "minimum": 0.01},
			"currency": {"type": "string", "enum": ["USD", "EUR", "GBP"]}
		},
		"required": ["transaction_id", "amount", "currency"]
	}`)

	schema, errs := jsonschema.FromJSON(externalSchema, nil)
	if len(errs) > 0 {
		fmt.Printf("  errors: %v\n", errs)
		return
	}

	// Use it directly — no need to rewrite in builder syntax.
	txn := map[string]interface{}{
		"transaction_id": "TXN-00012345",
		"amount":         99.99,
		"currency":       "USD",
	}
	if err := qf.Validate(txn, schema); err != nil {
		fmt.Printf("  unexpected: %v\n", err)
	} else {
		fmt.Println("  transaction validated: ok")
	}

	// You can also compile imported schemas for better performance.
	compiled := qf.Compile(schema)
	if err := qf.Validate(txn, compiled); err != nil {
		fmt.Printf("  unexpected: %v\n", err)
	} else {
		fmt.Println("  compiled validation:  ok")
	}
}

// strictModeExample shows how unsupported JSON Schema features are handled.
func strictModeExample() {
	fmt.Println("\n5. Handling Unsupported Features")
	fmt.Println("---------------------------------")

	// This schema uses $ref, which queryfy doesn't support.
	schemaWithRef := []byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"address": {"$ref": "#/$defs/Address"}
		}
	}`)

	// Default mode: unsupported features produce warnings but import continues.
	_, errs := jsonschema.FromJSON(schemaWithRef, nil)
	for _, e := range errs {
		fmt.Printf("  default: %s\n", e.Error())
	}

	// Strict mode: unsupported features produce errors.
	_, errs = jsonschema.FromJSON(schemaWithRef, &jsonschema.Options{
		StrictMode: true,
	})
	for _, e := range errs {
		fmt.Printf("  strict:  %s\n", e.Error())
	}
}

// metadataExample shows how custom extensions survive round-tripping.
func metadataExample() {
	fmt.Println("\n6. Preserving Custom Extensions")
	fmt.Println("---------------------------------")

	// JSON Schema with x- extension keywords.
	schemaJSON := []byte(`{
		"type": "string",
		"minLength": 1,
		"x-display-name": "Full Name",
		"x-placeholder": "Enter your name"
	}`)

	// Import with StoreUnknown to capture extensions as metadata.
	schema, _ := jsonschema.FromJSON(schemaJSON, &jsonschema.Options{
		StoreUnknown: true,
	})

	// Export with IncludeMeta to emit them back.
	exported, _ := jsonschema.ToJSON(schema, &jsonschema.ExportOptions{
		IncludeMeta: true,
	})

	fmt.Println("  exported with extensions preserved:")
	var pretty json.RawMessage = exported
	formatted, _ := json.MarshalIndent(pretty, "  ", "  ")
	fmt.Printf("  %s\n", formatted)
}
