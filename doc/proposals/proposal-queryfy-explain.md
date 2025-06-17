# Queryfy Explain() Method Development Proposal

## Executive Summary

This proposal outlines the implementation of an `Explain()` method for Queryfy's schema system. The feature would provide introspection capabilities that enhance debugging, documentation, and developer experience—core values of the Queryfy project.

## 1. Problem Statement

### 1.1 Current Limitations
- Schemas are opaque after creation, making debugging difficult
- No programmatic way to inspect validation rules
- Limited ability to generate documentation from schemas
- Developers must read source code to understand complex schemas

### 1.2 Market Research Alignment
Based on the market research findings, developers explicitly want:
- **Context-aware errors** that explain why validation failed
- **Self-documenting** validation systems
- **Better debugging tools** for complex validation logic

## 2. Proposed Solution

### 2.1 Core Feature: Explain() Method
Add an `Explain()` method to all schema types that returns a human-readable description of the validation rules.

```go
schema := builders.String().Email().MinLength(5).Required()
fmt.Println(schema.Explain())
// Output: "string (required): valid email, minimum length 5"
```

### 2.2 Advanced Feature: Structured Explanations
For complex use cases, provide a structured explanation format:

```go
explanation := schema.ExplainStructured()
// Returns a tree structure that can be serialized to JSON, used for documentation generation, etc.
```

## 3. Implementation Plan

### 3.1 Phase 1: Core Implementation (Week 1)
1. **Update Schema Interface** (Day 1)
   - Add `Explain() string` to the Schema interface in `schema.go`
   - Create helper utilities in new file `explain_helpers.go`

2. **Implement for Basic Types** (Days 2-3)
   - StringSchema: patterns, formats, length constraints
   - NumberSchema: ranges, integer/float requirements
   - BoolSchema: simple required/optional status
   - CustomSchema: display custom validator presence

3. **Implement for Container Types** (Days 4-5)
   - ArraySchema: element types, length constraints, uniqueness
   - ObjectSchema: field list, required fields summary

### 3.2 Phase 2: Advanced Schemas (Week 2)
4. **Complex Schema Types** (Days 1-2)
   - DateTimeSchema: formats, age constraints, date ranges
   - DependentSchema: condition descriptions, dependencies
   - TransformSchema: transformation pipeline description

5. **Composite Schemas** (Day 3)
   - AndSchema: "all of" explanations
   - OrSchema: "one of" explanations  
   - NotSchema: negation explanations

6. **Testing & Refinement** (Days 4-5)
   - Comprehensive unit tests
   - Integration examples
   - Performance validation

### 3.3 Phase 3: Enhanced Features (Optional, Week 3)
7. **Structured Explanations**
   - Define ExplanationTree structure
   - Implement ExplainStructured() method
   - JSON serialization support

8. **Documentation Generation**
   - Markdown generation from schemas
   - OpenAPI schema generation
   - Interactive schema explorer

## 4. Technical Design

### 4.1 Interface Changes
```go
// In schema.go
type Schema interface {
    Validate(value interface{}, ctx *ValidationContext) error
    Type() SchemaType
    Explain() string  // New method
}

// Optional enhancement
type ExplainableSchema interface {
    Schema
    ExplainStructured() *ExplanationTree
}
```

### 4.2 Helper Infrastructure
```go
// New file: explain_helpers.go
type ExplainBuilder struct {
    parts []string
}

func (e *ExplainBuilder) AddConstraint(name, value string)
func (e *ExplainBuilder) AddFormat(format string)
func (e *ExplainBuilder) Build() string
```

### 4.3 Example Implementations
```go
// StringSchema
func (s *StringSchema) Explain() string {
    eb := NewExplainBuilder(s.Type())
    eb.AddRequired(s.IsRequired())
    
    if s.minLength != nil {
        eb.AddConstraint("min length", strconv.Itoa(*s.minLength))
    }
    if s.formatType != "" {
        eb.AddFormat(s.formatType)
    }
    if len(s.enum) > 0 {
        eb.AddConstraint("enum", strings.Join(s.enum, ", "))
    }
    
    return eb.Build()
}

// ObjectSchema (more complex)
func (s *ObjectSchema) Explain() string {
    var required []string
    for name, schema := range s.fields {
        if isRequired(schema) {
            required = append(required, name)
        }
    }
    
    return fmt.Sprintf("object with %d fields (required: %s)",
        len(s.fields), strings.Join(required, ", "))
}
```

## 5. Benefits & Impact

### 5.1 Developer Experience Improvements
1. **Self-Documenting Schemas**: Schemas explain themselves
2. **Better Debugging**: Understand why validation failed
3. **Reduced Learning Curve**: New developers can inspect schemas
4. **API Documentation**: Auto-generate docs from schemas

### 5.2 Use Cases
1. **Development Time**
   ```go
   // Quick schema inspection during development
   fmt.Printf("User schema: %s\n", userSchema.Explain())
   ```

2. **Error Messages**
   ```go
   if err := qf.Validate(data, schema); err != nil {
       log.Printf("Validation failed. Expected: %s", schema.Explain())
   }
   ```

3. **API Documentation**
   ```go
   // Auto-generate API docs
   for endpoint, schema := range apiSchemas {
       docs.AddEndpoint(endpoint, schema.Explain())
   }
   ```

4. **Testing & Debugging**
   ```go
   // Verify schema configuration in tests
   assert.Contains(t, schema.Explain(), "email")
   ```

## 6. Implementation Considerations

### 6.1 Design Principles
1. **Clarity Over Brevity**: Explanations should be clear, not necessarily short
2. **Consistency**: Similar constraints should be explained similarly across types
3. **Extensibility**: Easy to add new explanation details without breaking changes
4. **Performance**: Explanation generation should be lightweight

### 6.2 Challenges & Solutions
| Challenge | Solution |
|-----------|----------|
| Repetitive implementation across 14+ files | Shared helper functions and base implementations |
| Complex nested schemas | Hierarchical explanation with indentation |
| Custom validators are opaque | Show presence and count of custom validators |
| Performance concerns | Lazy generation, caching for repeated calls |

### 6.3 Testing Strategy
1. **Unit Tests**: Each schema type's Explain() method
2. **Integration Tests**: Complex nested schemas
3. **Snapshot Tests**: Ensure explanations remain consistent
4. **Benchmark Tests**: Ensure minimal performance impact

## 7. Future Enhancements

### 7.1 Version 2 Features
1. **Explain Differences**: Compare two schemas
2. **Validation Trace**: Show which rules passed/failed
3. **Schema Visualization**: Generate diagrams
4. **IDE Integration**: Hover tooltips showing explanations

### 7.2 Ecosystem Integration
1. **OpenAPI Generation**: Convert schemas to OpenAPI specs
2. **GraphQL Integration**: Schema to GraphQL type definitions
3. **Documentation Plugins**: Auto-generate docs for various formats

## 8. Resource Requirements

### 8.1 Development Effort
- **Core Implementation**: 2 weeks (1 developer)
- **Testing & Documentation**: 1 week
- **Total**: 3 weeks for full implementation

### 8.2 Maintenance Impact
- Minimal ongoing maintenance
- New schema types automatically get Explain() via interface
- Helper functions reduce code duplication

## 9. Success Metrics

### 9.1 Quantitative
1. **Code Coverage**: 100% of schema types implement Explain()
2. **Performance**: <1ms explanation generation for typical schemas
3. **Documentation**: All examples updated to show Explain() usage

### 9.2 Qualitative
1. **Developer Feedback**: Easier debugging reported
2. **Adoption**: Explain() used in major projects
3. **Community**: Positive reception, feature requests for enhancements

## 10. Recommendation

### 10.1 Go/No-Go Decision
**Strong GO recommendation** based on:
- High benefit-to-effort ratio
- Direct alignment with Queryfy's developer experience focus
- Addresses explicit market needs identified in research
- Enhances Queryfy's differentiation from competitors

### 10.2 Priority Level
**HIGH PRIORITY** - This feature directly addresses core user needs and enhances Queryfy's main value proposition of making dynamic data validation developer-friendly.

### 10.3 Next Steps
1. **Approval**: Review and approve this proposal
2. **Design Review**: Finalize API design with team
3. **Implementation**: Begin Phase 1 development
4. **Community Preview**: Release beta for feedback
5. **Documentation**: Update all docs with new feature
6. **Release**: Include in next minor version (v0.2.0)

## Conclusion

The Explain() method represents a natural evolution of Queryfy's developer-first philosophy. By making schemas introspectable and self-documenting, we remove one more friction point in working with dynamic data validation. The implementation effort is reasonable, the benefits are clear, and the feature directly addresses needs identified in market research.

This enhancement would further establish Queryfy as the most developer-friendly validation library in the Go ecosystem.