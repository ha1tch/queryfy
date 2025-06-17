# Queryfy Schema-Struct Synchronization: Philosophy and Practice (Enhanced)

## Table of Contents

1. [The Core Challenge](#the-core-challenge)
2. [The Philosophy of Transvasing](#the-philosophy-of-transvasing)
3. [Multiple Schemas Per Entity](#multiple-schemas-per-entity)
4. [The Synchronization Strategy](#the-synchronization-strategy)
5. [Implementation Patterns](#implementation-patterns)
6. [Test-Time Verification](#test-time-verification)
7. [The Complete Data Journey](#the-complete-data-journey)
8. [Performance Analysis and Trade-offs](#performance-analysis-and-trade-offs)
9. [Why This Approach Over Alternatives](#why-this-approach-over-alternatives)
10. [Philosophical Clarity: Embracing Reality](#philosophical-clarity-embracing-reality)
11. [Future Directions](#future-directions)
12. [Conclusion: The Long Arc](#conclusion-the-long-arc)

---

## The Core Challenge

In Go development, we live in two worlds:

1. **The Dynamic World**: Where JSON arrives from external sources - messy, untyped, and unpredictable
2. **The Static World**: Where Go structs provide type safety, IDE support, and compile-time guarantees

The challenge isn't just moving data between these worlds - it's doing so while maintaining correctness, performance, and developer sanity.

### Why This Matters

Traditional approaches force uncomfortable choices:
- Direct unmarshal to structs: Panics on unexpected data
- Working with `map[string]interface{}`: Loses all type safety
- Manual validation after unmarshal: Redundant and error-prone

Queryfy introduces a third way: **controlled transvasing** - the careful transfer of data from dynamic to static form through a validation/transformation pipeline.

---

## The Philosophy of Transvasing

"Transvasing" - transferring liquid from one vessel to another - perfectly captures what we're doing with data. Like a chemist carefully transferring a solution through a filter, we're moving data from its raw form into its final container.

### Key Principles

1. **The data changes vessels, not just validation state**
   ```go
   // Not this: same data, different state
   rawData → [validate] → validatedData (same structure)
   
   // But this: transformed into new vessel
   rawData → [parse/transform] → shapedData → [transvase] → struct
   ```

2. **Validation is just ensuring transvasing is possible**
   - We don't validate for validation's sake
   - We validate because we need the data in a specific shape
   - If transvasing would fail, we want to know early

3. **The schema defines the transformation, not just the rules**
   - A schema is a specification for parsing
   - It describes how to create valid data, not just check it

---

## Multiple Schemas Per Entity

Real-world entities require different representations for different operations. Create operations need passwords while Updates don't; Query schemas use filter syntax rather than entity structure; Admin endpoints expose different fields than public APIs; Import operations need lenient validation while APIs demand strict validation. This is why we typically see:

```go
type UserSchemas struct {
    Create         Schema  // All fields required, includes password
    Update         Schema  // Most fields optional, no password
    PartialUpdate  Schema  // PATCH semantics
    Query          Schema  // Filter parameters, not entity fields
    AdminView      Schema  // Full data including sensitive fields
    PublicView     Schema  // Safe subset for public API
    Import         Schema  // Lenient validation for bulk imports
}
```

This pattern acknowledges that a single entity has multiple valid representations depending on context.

---

## The Synchronization Strategy

### Design-Time Organization and Declaration

```
entities/
├── user.go                    // type User struct {...}
├── user_schemas.go            // All User-related schemas
└── schema_test.go            // Compatibility tests
```

Schemas live alongside their corresponding structs, making relationships explicit:

```go
// user_schemas.go
package entities

func NewUserSchemas() *UserSchemas {
    // Shared field builders for consistency
    emailField := func(required bool) Schema {
        field := builders.Transform(
            builders.String().Email()
        ).Add(transformers.Lowercase())
        
        if required {
            return field.Required()
        }
        return field.Optional()
    }
    
    return &UserSchemas{
        Create: builders.Object().
            ForStruct(User{}).  // Links schema to struct
            Field("email", emailField(true)).
            Field("password", passwordField(true)),
            
        Update: builders.Object().
            ForStruct(User{}).
            Field("email", emailField(false)),
            // No password field in updates
            
        Query: builders.Object().
            // No ForStruct - doesn't map to User
            Field("email", builders.String().Optional()).
            Field("limit", builders.Number().Max(100)),
    }
}
```

### The ForStruct Annotation: How It Works

The `ForStruct()` method creates a critical link between schemas and structs:

```go
// Internal implementation concept
type ObjectSchema struct {
    fields       map[string]Schema
    targetStruct reflect.Type  // Set by ForStruct
}

func (s *ObjectSchema) ForStruct(v interface{}) *ObjectSchema {
    s.targetStruct = reflect.TypeOf(v)
    return s
}
```

#### What ForStruct Enables

1. **Type Information Storage**
   ```go
   // The schema now knows:
   // - Field names and types in the target struct
   // - JSON tags for field mapping
   // - Whether fields are pointers (optional)
   ```

2. **Compatibility Verification**
   ```go
   func (s *ObjectSchema) VerifyStruct(target interface{}) error {
       targetType := reflect.TypeOf(target).Elem()
       
       // For each schema field:
       for fieldName, fieldSchema := range s.fields {
           // Find corresponding struct field
           structField, found := targetType.FieldByName(fieldName)
           if !found {
               // Check JSON tag
               structField, found = findByJSONTag(targetType, fieldName)
           }
           
           if !found {
               return fmt.Errorf("schema field %q has no struct field", fieldName)
           }
           
           // Verify type compatibility
           schemaType := fieldSchema.GetOutputType()
           if !isAssignable(schemaType, structField.Type) {
               return fmt.Errorf("field %q: schema produces %v, struct expects %v",
                   fieldName, schemaType, structField.Type)
           }
       }
       return nil
   }
   ```

3. **Future ToStruct Optimization**
   ```go
   // With ForStruct, ToStruct can be optimized:
   func (qf *Queryfy) ToStruct(data interface{}, target interface{}) error {
       schema := getSchemaFor(target) // Retrieved via ForStruct mapping
       
       // Direct field mapping without full reflection scan
       return optimizedMapping(data, target, schema.fieldMappings)
   }
   ```

---

## Test-Time Verification

Instead of runtime checks that impact performance, we verify compatibility at test time. This can be done through generated tests or manual patterns:

```go
// schema_test.go
func TestSchemaStructCompatibility(t *testing.T) {
    schemas := NewUserSchemas()
    
    // Verify each schema that maps to User
    for name, schema := range map[string]Schema{
        "Create": schemas.Create,
        "Update": schemas.Update,
        // Note: Query not included - doesn't map to User
    } {
        t.Run(name, func(t *testing.T) {
            var user User
            if err := schema.VerifyStruct(&user); err != nil {
                t.Fatal(err)
                // Detailed errors like:
                // Field "email": schema produces string, struct expects int
                // Field "phone": schema field has no corresponding struct field
            }
        })
    }
}
```

### What Gets Verified

1. **Type Compatibility**: Can schema output be assigned to struct field? Are numeric types compatible?
2. **Field Mapping**: Does every schema field have a struct target? Are JSON tags properly mapped?
3. **Transformation Compatibility**: Will transformed output match struct types?

---

## The Complete Data Journey

```go
// Stage 1 & 2: Reception and Transformation
rawJSON := receiveFromAPI()
schema := schemas.User.Create
cleanData, err := schema.ValidateAndTransform(rawJSON)
// cleanData now has validated structure, transformed values, and correct types

// Stage 3 & 4: Inspection and Transvasing  
email := cleanData["email"].(string)  // Safe - schema guarantees string
cleanData["createdAt"] = time.Now()   // Add computed fields if needed

var user User
err = qf.ToStruct(cleanData, &user)
// Success guaranteed if schema.VerifyStruct() passed in tests

// Stage 5: Business Logic
user.ID = generateID()
err = db.Save(&user)
```

Each stage serves a specific purpose in moving from untrusted external data to trusted internal structures.

---

## Performance Analysis and Trade-offs

### When We Maintain Zero Allocations

Based on our benchmarking sessions, Queryfy + Superjsonic maintains zero allocations during:

1. **Token parsing phase**: Reading and tokenizing JSON
2. **Structural validation**: Checking JSON structure matches schema
3. **Simple type validation**: Verifying strings, numbers, booleans

### When We Allocate Memory

Memory allocation occurs during:

1. **Transformation phase**: Creating new cleaned data structures
   ```go
   // Example: Transforming email
   "  JOHN@EXAMPLE.COM  " → "john@example.com"  // New string allocated
   ```

2. **Map/slice creation for results**: Building the cleaned data map
   ```go
   cleanData := make(map[string]interface{})  // Allocation
   ```

3. **ToStruct phase**: Converting to final struct form

### Performance Characteristics by Payload Size

**Small payloads (<10KB)**: 
- Memory overhead negligible
- Transformation cost minimal
- Suitable for REST APIs, microservices

**Medium payloads (10KB-1MB)**:
- Memory temporarily doubles during transformation
- Still performant for most use cases
- Consider streaming for arrays

**Large payloads (>1MB)**:
- Memory pressure becomes significant
- Current research into pooling strategies
- Batch processing patterns being developed

### Future Optimizations: Batch Processing Research

We're actively researching strategies for maintaining near-zero allocations even during transformation of large datasets. The key insight is that batch processing has different characteristics than single-document processing:

```go
// Concept: ForStructBatch for high-volume processing
schema := builders.Object().
    ForStructBatch(User{}, BatchOptions{
        PoolSize: 1000,
        ReuseAllocations: true,
    })

// This would enable:
processor := qf.NewBatchProcessor(schema)
for _, jsonBatch := range hugeDataset {
    users := processor.ProcessBatch(jsonBatch)
    // Reuses memory allocations across batches
}
```

#### The Batch Processing Strategy

When processing millions of small JSON documents (think log ingestion, event streams, bulk imports), the allocation pattern changes:

1. **Pre-allocate transformation buffers** based on expected document size
2. **Reuse map structures** - clear and refill rather than allocate new
3. **Pool string buffers** for transformations like lowercase/trim
4. **Amortize allocation cost** across thousands of documents

This research is ongoing because we want to ensure:
- The API remains simple for common cases
- Batch optimizations don't complicate single-document usage  
- Memory pooling doesn't introduce concurrency issues
- The benefits justify the additional complexity

We're learning from Superjsonic's successful pooling strategies and exploring how to apply similar principles to the transformation phase.

---

## Why This Approach Over Alternatives

### Why Not Generate Schemas from Structs?

```go
// The attempt:
type User struct {
    Email string `json:"email" validate:"email,required"`
}
```

**The problems**: Struct tags can't express transformations (how do you tag "lowercase and trim"?), can't handle messy real-world data (API sends "25" for an int field), and can't represent different operations (create needs required email, update needs optional).

### Why Not Fix Data After Unmarshal?

```go
// The attempt:
json.Unmarshal(data, &user)
user.Email = strings.ToLower(user.Email)
```

**The problems**: Unmarshal might panic on wrong types, transformation logic gets scattered across codebase, and you can't validate until after potentially corrupting your structs.

### Why Not Separate Validation and Transformation?

```go
// The attempt:
validate(data) → transform(data) → unmarshal(data)
```

**The problems**: Multiple passes hurt performance, validation and transformation are naturally intertwined (valid email includes normalization), and error messages can't suggest what transformation would fix issues.

**The key insight**: Validation is never the end goal. The goal is getting data into a shape you can use. Queryfy acknowledges this by treating validation and transformation as one operation.

---

## Philosophical Clarity: Embracing Reality

### The False Dichotomy

Many systems try to maintain rigid separation:
- "Schemas should know nothing about structs"
- "Validation should never modify data"
- "Dynamic and static typing shouldn't mix"

This ideological purity creates practical problems.

### Queryfy's Pragmatic Philosophy

**We embrace the natural tension** between dynamic JSON and static structs. This tension isn't a problem to solve - it's a reality to work with.

1. **Schemas know about structs when useful** (ForStruct)
   - But structs don't depend on schemas
   - One-way coupling is intentional and healthy

2. **Validation and transformation are one operation**
   - Because that's what you actually need
   - Separating them is artificial

3. **Multiple representations are normal**
   - One struct, many schemas
   - Different views for different contexts

### Working with Reality

The real world presents us with messy, inconsistent data where:
- APIs send numbers as strings
- Dates come in 15 formats
- Phone numbers are chaos
- Required fields are sometimes missing

The philosophy isn't about claiming to have all the answers. We're exploring practical solutions to real problems, learning from each implementation, and adjusting our approach based on what we discover. This humility is built into Queryfy's design - we provide escape hatches, multiple approaches, and acknowledge that different scenarios need different solutions.

---

## Future Directions

### Generic Schema Builders (Go 1.18+)
```go
schema := builders.ObjectFor[User]().
    Field("email", builders.String().Email())
// Compile error if "email" doesn't exist in User
```

### Automatic Schema Generation with Enhancement
```go
baseSchema := qf.GenerateSchema(User{})
// Then enhance what tags can't express:
schema := baseSchema.
    FieldTransform("email", transformers.Lowercase()).
    FieldValidation("phone", validators.PhoneNumber("US"))
```

### IDE Integration
- Plugin that shows schema-struct mappings
- Warnings for incompatible changes
- Auto-generate compatibility tests

### Performance Optimizations
- Code generation for ToStruct to avoid reflection
- Streaming transvasing for large datasets
- Parallel processing for array fields

---

## Conclusion: The Long Arc

In the long arc of Queryfy's data processing philosophy, schema-struct synchronization represents the final bridge between two worlds that have been artificially separated in most validation libraries.

### The Journey We Enable

1. **Receive** messy, real-world data
2. **Parse** it through validation into clean form
3. **Transform** it during parsing for efficiency
4. **Verify** the result matches expectations
5. **Transvase** into type-safe structures
6. **Process** with confidence in business logic

### The Key Innovation

By treating validation as parsing and transformation as part of that parsing, we've created a system where:
- **Schemas are active participants** in data shaping, not passive validators
- **Tests guarantee production safety** through compile-time and test-time verification
- **Multiple schemas per entity** acknowledge real-world complexity
- **Transvasing is safe** because compatibility is verified upstream

### The Philosophy Realized

"Parse, don't validate" reaches its full expression when the parsing pipeline ends with type-safe data in proper Go structs. We haven't just validated the data - we've transformed it, cleaned it, and placed it safely in its final vessel.

This isn't just about making JSON handling safer or faster. It's about bringing the full power of Go's type system to bear on real-world data processing while acknowledging that the real world is messy, inconsistent, and changeable.

### The Final Insight

In traditional approaches, you validate data to prove it matches your structs. In Queryfy's approach, you parse data into shapes that fit your structs. This inversion - from checking to creating - is what makes the system both powerful and ergonomic.

The long arc bends toward safety, but it gets there through transformation, not restriction. By embracing the reality of messy data rather than wishing it away, Queryfy provides a practical path from the chaos of external APIs to the safety of Go's type system.

That's the Queryfy way: meet data where it is, shape it into what you need, and deliver it safely to its destination.