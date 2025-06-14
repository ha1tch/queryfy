# Queryfy: Compose at Build-Time, Validate at Run-Time

## The Fundamental Principle

Queryfy embodies a design philosophy that aims to ease the eternal tension between flexibility and predictability in software systems. At its core, this philosophy recognizes that **composition** and **configuration** are not opposing forces, but rather two sides of the same coin that manifest at different phases of your application's lifecycle.

## The Two-Phase Model

### Phase 1: Build-Time Composition (Maximum Flexibility)

During the build phase, Queryfy provides unlimited flexibility through composable builders. This is where creativity and adaptability reign:

```go
// Compose your validation logic with complete freedom
schema := builders.Object().
    Field("email", builders.String().Email().Required()).
    Field("age", builders.Number().Min(18).Max(120)).
    Field("preferences", builders.Object().
        Field("notifications", builders.Bool()).
        Field("theme", builders.String().Enum("light", "dark")))

// Compose behaviors through transformation
emailSchema := builders.Transform(
    builders.String().Email()
).Add(transformers.Lowercase()).Add(transformers.Trim())

// Compose complex logic through combinators
contactSchema := builders.Or(
    builders.String().Email(),
    builders.String().Pattern(`^\+?[1-9]\d{9,14}$`)
)
```

### Phase 2: Run-Time Validation (Complete Determinism)

Once composed, schemas become immutable validators with predictable, deterministic behavior:

```go
// At runtime, behavior is fixed and predictable
err := qf.Validate(userData, schema)  // Same input → Same output, always

// No mutations, no surprises
// The schema cannot change during validation
// The validation process is pure and side-effect free
```

## Why This Matters

### 1. **Mental Model Clarity**

Developers can think in two distinct modes:
- **Design mode**: "How do I compose the behavior I need?"
- **Runtime mode**: "What will happen when this validates?"

This separation reduces cognitive load and makes systems easier to reason about.

### 2. **Type-Safe Flexibility**

Composition at build-time allows the Go compiler to catch errors:

```go
// This won't compile - Email() is not available on Number
schema := builders.Number().Email()  // ❌ Compile error

// The compiler guides you to valid compositions
schema := builders.String().Email()  // ✅ Type-safe
```

### 3. **Performance Through Invariance**

Because schemas are invariant at runtime, Queryfy can optimize aggressively:
- Regex patterns compiled once
- Validation paths predetermined
- No runtime decision trees
- Predictable memory usage

## The Composition Patterns

### Pattern 1: Layered Composition

```go
// Base layer
baseUser := builders.Object().
    Field("id", builders.String().UUID()).
    Field("createdAt", builders.DateTime())

// Enhancement layer
activeUser := baseUser.
    Field("email", builders.String().Email().Required()).
    Field("lastLogin", builders.DateTime().Required())

// Specialization layer
adminUser := activeUser.
    Field("permissions", builders.Array().Required()).
    Field("auditLog", builders.Array())
```

### Pattern 2: Behavioral Composition

```go
// Compose behaviors, not just structure
validatedString := builders.String().
    Transform(transformers.Trim()).
    Transform(transformers.Lowercase()).
    MinLength(3).
    MaxLength(50)

// Each method adds behavior, creating a pipeline
// The final schema represents the composition of all behaviors
```

### Pattern 3: Conditional Composition

```go
// Even conditional logic is composed at build-time
schema := builders.Object().WithDependencies().
    Field("userType", builders.String().Enum("person", "company")).
    DependentField("firstName",
        builders.Dependent("firstName").
            When(builders.WhenEquals("userType", "person")).
            Then(builders.String().Required())).
    DependentField("companyName",
        builders.Dependent("companyName").
            When(builders.WhenEquals("userType", "company")).
            Then(builders.String().Required()))

// The conditions are evaluated at runtime,
// but the structure is fixed at build-time
```

## Architectural Implications

### The Four Layers

```
┌─────────────────────────────┐
│   Declaration Layer         │ ← What to validate
├─────────────────────────────┤
│   Composition Layer         │ ← How to build it
├─────────────────────────────┤
│   Behavior Layer            │ ← Fixed at build-time
├─────────────────────────────┤
│   Execution Layer           │ ← Deterministic runtime
└─────────────────────────────┘
```

Each layer has a specific responsibility:
- **Declaration**: Define the structure of your data
- **Composition**: Combine behaviors and constraints
- **Behavior**: The immutable result of composition
- **Execution**: Pure, predictable validation

### Flexibility Through Immutability

Counter-intuitively, immutability at runtime enables greater flexibility at build-time:

```go
// Because schemas are immutable, they can be safely shared
var (
    emailSchema = builders.String().Email().Required()
    phoneSchema = builders.String().Pattern(`^\+?[1-9]\d{9,14}$`)
)

// Reuse without fear of mutation
userSchema := builders.Object().
    Field("primaryEmail", emailSchema).
    Field("secondaryEmail", emailSchema).  // Safe reuse
    Field("phone", phoneSchema)
```

## The Configuration-Composition Continuum

### It's Not Either/Or

The true power emerges when we recognize that configuration and composition work together, not against each other. Configuration selects between pre-composed behaviors:

```go
// Pre-composed schemas for different API versions
var (
    // API v1.0 - Basic validation
    schemaV1 = builders.Object().
        Field("userId", builders.String().Required()).
        Field("action", builders.String().Required())
    
    // API v1.1 - Added timestamp validation
    schemaV1_1 = builders.Object().
        Field("userId", builders.String().Required()).
        Field("action", builders.String().Required()).
        Field("timestamp", builders.DateTime().ISO8601().Required())
    
    // API v2.0 - Enhanced with audit fields
    schemaV2 = builders.Object().
        Field("userId", builders.String().UUID().Required()).
        Field("action", builders.String().Enum("create", "update", "delete").Required()).
        Field("timestamp", builders.DateTime().ISO8601().Required()).
        Field("metadata", builders.Object().Optional())
)

// Configuration selects which composition to use
func getSchemaForRequest(r *http.Request) Schema {
    apiVersion := os.Getenv("API_VERSION")
    
    // Progressive rollout based on configuration
    if featureFlag.IsEnabled("api_v2", r.Header.Get("X-User-ID")) {
        return schemaV2
    }
    
    switch apiVersion {
    case "1.1":
        return schemaV1_1
    case "2.0":
        return schemaV2
    default:
        return schemaV1
    }
}
```

### Configuration as Composition Selection

Configuration doesn't create behavior—it selects from pre-composed behaviors:

```go
// Compose all possible behaviors at build-time
type ValidationStrategy struct {
    Strict    Schema
    Lenient   Schema  
    Migration Schema  // Accepts both old and new formats
}

var userValidation = ValidationStrategy{
    Strict: builders.Object().
        Field("email", builders.String().Email().Required()).
        Field("phone", builders.String().Pattern(`^\+\d{11,14}$`).Required()),
    
    Lenient: builders.Object().
        Field("email", builders.String().Email().Required()).
        Field("phone", builders.String().Optional()),
    
    Migration: builders.Object().
        Field("email", builders.String().Email().Required()).
        Field("phone", builders.Or(
            builders.String().Pattern(`^\+\d{11,14}$`),    // New format
            builders.String().Pattern(`^\d{10}$`),          // Legacy format
        )),
}

// Configuration determines which composition to use
func validateUser(data map[string]interface{}) error {
    mode := config.GetString("validation.mode")
    
    var schema Schema
    switch mode {
    case "strict":
        schema = userValidation.Strict
    case "migration":
        schema = userValidation.Migration
    default:
        schema = userValidation.Lenient
    }
    
    return qf.Validate(data, schema)
}
```

### Progressive Feature Adoption

Configuration enables gradual rollout of new validation rules:

```go
// Feature flags control which validations are active
type FeatureSchemas struct {
    schemas map[string]Schema
    mu      sync.RWMutex
}

func (fs *FeatureSchemas) Register(feature string, schema Schema) {
    fs.mu.Lock()
    defer fs.mu.Unlock()
    fs.schemas[feature] = schema
}

func (fs *FeatureSchemas) GetActive() Schema {
    fs.mu.RLock()
    defer fs.mu.RUnlock()
    
    // Compose active features based on configuration
    active := builders.Object()
    
    for feature, schema := range fs.schemas {
        if featureFlag.IsEnabled(feature) {
            // Merge schemas for enabled features
            active = mergeSchemas(active, schema)
        }
    }
    
    return active
}

// Usage
features := &FeatureSchemas{}

// Register feature schemas
features.Register("enhanced_validation", 
    builders.Object().Field("score", builders.Number().Min(0).Max(100)))

features.Register("ml_predictions",
    builders.Object().Field("confidence", builders.Number().Min(0).Max(1)))

// Configuration determines active features
activeSchema := features.GetActive()
```

## Advanced Patterns

### Dynamic Schema Selection

Configuration can dynamically select schemas based on runtime context:

```go
// Schema registry with version management
type SchemaRegistry struct {
    versions map[string]map[string]Schema  // domain -> version -> schema
}

func (sr *SchemaRegistry) GetSchema(domain, clientVersion string) Schema {
    // Default version from configuration
    defaultVersion := config.GetString(fmt.Sprintf("%s.default_version", domain))
    
    // Client version override
    if clientVersion != "" && sr.supportsVersion(domain, clientVersion) {
        return sr.versions[domain][clientVersion]
    }
    
    // Gradual rollout based on percentage
    rolloutPercent := config.GetInt(fmt.Sprintf("%s.v2_rollout_percent", domain))
    if shouldRollout(rolloutPercent) {
        if schema, ok := sr.versions[domain]["v2"]; ok {
            return schema
        }
    }
    
    return sr.versions[domain][defaultVersion]
}

// Pre-composed schemas registered at startup
func initSchemas(registry *SchemaRegistry) {
    // User domain schemas
    registry.Register("user", "v1", builders.Object().
        Field("name", builders.String().Required()).
        Field("email", builders.String().Email()))
    
    registry.Register("user", "v2", builders.Object().
        Field("name", builders.String().Required()).
        Field("email", builders.String().Email().Required()).
        Field("emailVerified", builders.Bool()).
        Field("profile", builders.Object()))
    
    // Order domain schemas  
    registry.Register("order", "v1", legacyOrderSchema)
    registry.Register("order", "v2", modernOrderSchema)
}
```

### Environment-Aware Composition

Different environments may require different validation strategies:

```go
// Compose environment-specific behaviors
type EnvironmentSchemas struct {
    Development Schema
    Staging     Schema
    Production  Schema
}

func BuildEnvironmentSchemas() EnvironmentSchemas {
    base := builders.Object().
        Field("id", builders.String().Required()).
        Field("timestamp", builders.DateTime())
    
    return EnvironmentSchemas{
        // Development: More lenient
        Development: base.
            Field("debug", builders.Object().Optional()).
            Field("testData", builders.Bool().Optional()),
        
        // Staging: Close to production
        Staging: base.
            Field("version", builders.String().Required()),
        
        // Production: Strict validation
        Production: base.
            Field("version", builders.String().Required()).
            Field("checksum", builders.String().Required()).
            Custom(validateIntegrity),
    }
}

// Configuration selects environment
func GetSchemaForEnvironment() Schema {
    env := os.Getenv("APP_ENV")
    schemas := BuildEnvironmentSchemas()
    
    switch env {
    case "development":
        return schemas.Development
    case "staging":
        return schemas.Staging
    default:
        return schemas.Production
    }
}
```

### A/B Testing Validation Rules

Configuration enables testing different validation strategies:

```go
// Compose alternative validation strategies
type ABTestSchemas struct {
    Control    Schema  // Current validation
    Variant    Schema  // New validation to test
    Metrics    *ValidationMetrics
}

func (ab *ABTestSchemas) ValidateWithABTest(data interface{}, userID string) error {
    // Configuration determines test participation
    inVariant := abtest.IsUserInVariant("strict_validation_test", userID)
    
    var schema Schema
    var group string
    
    if inVariant {
        schema = ab.Variant
        group = "variant"
    } else {
        schema = ab.Control
        group = "control"
    }
    
    start := time.Now()
    err := qf.Validate(data, schema)
    duration := time.Since(start)
    
    // Track metrics for analysis
    ab.Metrics.Record(group, err == nil, duration)
    
    return err
}

// Setup A/B test
abTest := &ABTestSchemas{
    Control: builders.Object().
        Field("email", builders.String()),  // Optional email
        
    Variant: builders.Object().
        Field("email", builders.String().Email().Required()),  // Required valid email
        
    Metrics: NewValidationMetrics(),
}
```

## The Complete Picture

### Three Levels of Flexibility

1. **Build-Time Composition**: Create all possible behaviors
2. **Configuration Selection**: Choose which behaviors to activate
3. **Runtime Execution**: Deterministic validation

```
┌─────────────────────────────┐
│   Configuration Layer       │ ← Selects behaviors
├─────────────────────────────┤
│   Composition Layer         │ ← Defines behaviors
├─────────────────────────────┤
│   Validation Layer          │ ← Executes behaviors
└─────────────────────────────┘
```

### Benefits of This Approach

1. **All behaviors are tested**: Since schemas are pre-composed, they can be tested
2. **No runtime surprises**: Configuration only selects, never creates
3. **Progressive adoption**: Roll out changes gradually with confidence
4. **Environment flexibility**: Different rules for different contexts
5. **Performance**: No schema building overhead at runtime

### Anti-Patterns to Avoid

```go
// ❌ Don't build schemas from configuration at runtime
func badPattern(config map[string]interface{}) Schema {
    schema := builders.Object()
    for field, rules := range config {
        // This creates untested, unpredictable schemas
        schema.Field(field, buildFromRules(rules))
    }
    return schema
}

// ✅ Do select from pre-built schemas
func goodPattern(config map[string]interface{}) Schema {
    schemaName := config["schema"].(string)
    return schemaRegistry.Get(schemaName)
}
```

## The Philosophy in Practice

### What This Enables

1. **Predictable Systems**: Once deployed, behavior is guaranteed
2. **Testable Logic**: Pure functions with no side effects
3. **Composable Libraries**: Build higher-level abstractions safely
4. **Clear Boundaries**: Build-time vs run-time responsibilities

### What This Prevents

1. **Runtime Surprises**: No validation rules changing mid-execution
2. **Configuration Errors**: Caught at compile-time, not production
3. **Mutation Bugs**: Immutable schemas can't be accidentally modified
4. **Performance Degradation**: No runtime schema building overhead

## Conclusion

The principle of "Compose at Build-Time, Validate at Run-Time" doesn't mean abandoning configuration—it means using configuration wisely. Configuration becomes a selection mechanism for choosing between pre-composed, tested behaviors rather than a way to create behaviors dynamically.

This approach provides:
- **Flexibility through composition**: Build all the variations you need
- **Safety through selection**: Configuration only chooses what already exists
- **Evolution through versioning**: Gradually roll out new behaviors
- **Confidence through testing**: All possible behaviors can be tested

By combining the power of composition with the flexibility of configuration, Queryfy enables systems that are simultaneously:
- **Flexible**: Adapt to different contexts and requirements
- **Predictable**: All behaviors are known and tested
- **Evolvable**: Changes can be rolled out progressively
- **Performant**: No runtime schema construction overhead

This is not about choosing between configuration and code—it's about using each tool for what it does best. Configuration excels at runtime selection; composition excels at behavior definition. Together, they create systems that are both powerful and manageable.

The future of validation isn't in more complex configuration languages or more dynamic runtime behavior—it's in better composition tools that make it easy to build the exact deterministic behaviors you need, combined with simple configuration that selects the right behavior for the right context.
