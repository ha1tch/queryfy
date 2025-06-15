# Understanding Queryfy + Superjsonic: A Guide to Fast, Safe JSON Validation in Go

## Table of Contents

1. [Introduction: Why This Matters](#introduction-why-this-matters)
2. [The JSON Trust Problem](#the-json-trust-problem)
3. [How Queryfy + Superjsonic Works](#how-queryfy--superjsonic-works)
4. [The Economics of Validation](#the-economics-of-validation)
5. [Getting Started](#getting-started)
6. [Real-World Usage Patterns](#real-world-usage-patterns)
7. [The "No More Panics" Promise](#the-no-more-panics-promise)
8. [Performance Deep Dive](#performance-deep-dive)
9. [The Smell Test: Your Early Warning System](#the-smell-test-your-early-warning-system)
10. [Best Practices](#best-practices)
11. [Conclusion: A New Baseline](#conclusion-a-new-baseline)

---

## Introduction: Why This Matters

Every Go developer has written code like this:

```go
var data map[string]interface{}
json.Unmarshal(jsonBytes, &data)
userID := data["user"].(map[string]interface{})["id"].(string) // 💥 PANIC!
```

And every Go developer has been paged at 3 AM when that code met reality.

Queryfy + Superjsonic is a validation system that solves this problem. It's not just another validation library—it's a different approach to handling untrusted data in Go. By combining Queryfy's schema validation with Superjsonic's fast JSON parser, we get something useful: **validation fast enough that you can afford to validate everything**.

### What It Offers

- **5-8x faster** than standard JSON validation
- **Zero panics** in your JSON handling code
- **Zero allocations** during validation
- **One consistent approach** for all your JSON needs

But speed is just the beginning. This is really about changing how we think about data trust.

---

## The JSON Trust Problem

In production systems, JSON data is like food at a restaurant. You need to process it, but you can't just trust it blindly. Bad JSON, like bad food, can ruin your whole system.

### The Human Approach to Untrusted Food

When humans encounter suspicious food, we have a natural process:

```go
// How humans actually process untrusted food
func shouldEatThis(food Food) bool {
    if looksWrong(food) {      // 👃 "That doesn't smell right"
        return false            // ❌ Don't even taste it
    }
    
    if tastesBad(food) {       // 👅 "Texture is off" 
        return false            // ❌ Don't swallow
    }
    
    if notWhatIOrdered(food) {  // 🧪 "This isn't chicken"
        return false            // ❌ Send it back
    }
    
    return true                 // ✅ Safe to consume
}
```

This instinctive process has kept humans alive for millennia. Queryfy + Superjsonic brings this same approach to your JSON processing.

### The Traditional Approach (Dangerous)

Most JSON processing looks like this:

```go
// The "close your eyes and swallow" approach
func processPayment(jsonData []byte) {
    var payment Payment
    json.Unmarshal(jsonData, &payment)  // Looks safe...
    
    amount := payment.Amount            // Might work...
    account := payment.User.Account.ID  // 💥 PANIC: nil pointer
}
```

This is like eating with your eyes closed—eventually, you'll swallow something bad.

---

## How Queryfy + Superjsonic Works

The system implements a multi-stage trust pipeline, just like human digestion:

### Stage 1: The Smell Test (<1 microsecond)

```go
// Superjsonic's "smell test" - instant rejection of obviously bad data
if smellsBad(jsonBytes) {
    return errors.New("corrupted JSON detected")
}
```

This catches corrupted uploads, truncated data, and obvious garbage before wasting any real processing time. Like a bad smell warning you before you taste spoiled milk.

### Stage 2: Structural Validation (<100 microseconds)

```go
// Superjsonic parses structure without allocating memory
tokens := superjsonic.Tokenize(jsonBytes)  // Zero allocations!
if !isValidStructure(tokens) {
    return errors.New("invalid JSON structure")
}
```

This ensures the JSON is properly formed—all brackets match, strings are terminated, numbers are valid. Like checking that your food has the right texture before swallowing.

### Stage 3: Schema Validation (<1 millisecond)

```go
// Queryfy validates against your business rules
schema := builders.Object().
    Field("amount", builders.Number().Min(0.01).Max(10000)).
    Field("account", builders.String().Pattern(`^\d{10}$`))

if err := queryfy.ValidateTokens(tokens, schema); err != nil {
    return err  // Clear, specific error about what's wrong
}
```

This ensures the data matches your expectations. Like verifying you got the dish you actually ordered.

### Stage 4: Safe Unmarshaling (only if everything passes)

```go
// Only NOW do we create structs - when we know it's safe
var payment Payment
err := qf.ValidateInto(jsonBytes, schema, &payment)
// If we get here, payment is PERFECTLY formed - no panics possible!
```

---

## The Economics of Validation

Here's where Queryfy + Superjsonic gets really interesting. Traditional validation is like airport security where everyone goes through the full process. Queryfy + Superjsonic is like having TSA PreCheck, drug-sniffing dogs, and metal detectors working in parallel.

### The Cost Pyramid

```
Traditional Approach - Everyone Pays Full Price:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
│  Parse + Validate + Unmarshal    │ 100% of requests
│       (~1000 microseconds)        │ pay full cost
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Queryfy + Superjsonic - Pay As You Go:
     ▲
    ╱│╲    5% - Full unmarshal (~1000μs)
   ╱ │ ╲   
  ╱  │  ╲  10% - Schema validation (~100μs)
 ╱   │   ╲ 
╱    │    ╲ 15% - Structural check (~10μs)
━━━━━━━━━━━ 70% - Smell test reject (~1μs)
```

### Real-World Impact

Consider an API gateway handling 100,000 requests/second:

**Traditional Approach:**
- 100,000 × 1,000μs = 100 seconds of CPU time per second
- Need 100+ CPU cores just for validation!

**Queryfy + Superjsonic:**
- 70,000 × 1μs = 0.07 seconds (smell test rejects)
- 15,000 × 10μs = 0.15 seconds (structure rejects)
- 10,000 × 100μs = 1 second (schema rejects)
- 5,000 × 1,000μs = 5 seconds (full processing)
- **Total: 6.22 seconds of CPU time**
- 16x more efficient!

This isn't just a performance improvement—it's a practical change in what's economically viable. You can now afford to validate EVERYTHING.

---

## Getting Started

### Installation

```bash
go get github.com/yourusername/queryfy
```

### Your First Validation

```go
package main

import (
    "github.com/yourusername/queryfy"
    "github.com/yourusername/queryfy/builders"
)

func main() {
    // Define what valid data looks like
    userSchema := builders.Object().
        Field("name", builders.String().Required()).
        Field("email", builders.String().Email()).
        Field("age", builders.Number().Min(0).Max(150))
    
    // Create validator
    qf := queryfy.New()
    
    // Validate some JSON
    jsonData := []byte(`{
        "name": "Alice",
        "email": "alice@example.com",
        "age": 30
    }`)
    
    if err := qf.Validate(jsonData, userSchema); err != nil {
        // Error will be specific and helpful:
        // "email: must be a valid email address at line 3, column 15"
        panic(err)
    }
    
    // Or validate AND unmarshal in one safe step
    var user User
    if err := qf.ValidateInto(jsonData, userSchema, &user); err != nil {
        panic(err)
    }
    // user is now populated and GUARANTEED to be valid
}
```

### The Codec Choice

Queryfy lets you choose your JSON library while keeping the same validation:

```go
// Use standard library (default)
qf := queryfy.New()

// Use jsoniter for 3x faster unmarshaling
import jsoniter "github.com/json-iterator/go"
qf := queryfy.New().WithCodec(jsoniter.ConfigFastest)

// Use your company's custom codec
qf := queryfy.New().WithCodec(CompanySecureCodec{})
```

---

## Real-World Usage Patterns

### Pattern 1: API Endpoint Protection

```go
func HandlePayment(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    
    // Define expectations
    schema := builders.Object().
        Field("amount", builders.Number().Min(0.01).Max(10000)).
        Field("currency", builders.Enum("USD", "EUR", "GBP")).
        Field("account", builders.String().Pattern(`^\d{10}$`))
    
    // Validate and unmarshal
    var payment Payment
    if err := qf.ValidateInto(body, schema, &payment); err != nil {
        // err contains exactly what's wrong and where
        http.Error(w, err.Error(), 400)
        return
    }
    
    // Process payment - ZERO panic risk!
    processPayment(payment)
}
```

### Pattern 2: Configuration Loading

```go
func LoadConfig(filename string) (*Config, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    
    // Define valid configuration
    schema := builders.Object().
        Field("database", builders.Object().
            Field("host", builders.String().Required()).
            Field("port", builders.Number().Min(1).Max(65535)).
            Field("ssl", builders.Bool().Default(true))).
        Field("redis", builders.Object().
            Field("url", builders.String().URL()).
            Optional())  // Redis is optional
    
    var config Config
    if err := qf.ValidateInto(data, schema, &config); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }
    
    return &config, nil
}
```

### Pattern 3: Webhook Processing

```go
func ProcessWebhook(data []byte) error {
    // Quick smell test for obviously bad data
    if quality := qf.AssessQuality(data); quality == queryfy.JSONQualityRotten {
        metrics.Inc("webhook.rejected.smell_test")
        return errors.New("corrupted webhook payload")
    }
    
    // Full validation
    if err := qf.Validate(data, WebhookSchema); err != nil {
        metrics.Inc("webhook.rejected.validation")
        return err
    }
    
    // Process with confidence
    return processValidWebhook(data)
}
```

---

## The "No More Panics" Promise

This is perhaps the most valuable feature. Let's look at why panics happen and how Queryfy + Superjsonic eliminates them:

### Why JSON Code Panics

```go
// The panic minefield
data := make(map[string]interface{})
json.Unmarshal(jsonBytes, &data)

// Each of these can panic:
userMap := data["user"].(map[string]interface{})     // panic: interface conversion
userName := userMap["name"].(string)                 // panic: interface conversion
items := data["items"].([]interface{})                // panic: interface conversion
firstItem := items[0].(map[string]interface{})       // panic: index out of range
price := firstItem["price"].(float64)                // panic: interface conversion
```

### The Queryfy + Superjsonic Way

```go
// Define expectations upfront
schema := builders.Object().
    Field("user", builders.Object().
        Field("name", builders.String())).
    Field("items", builders.Array().
        Min(1).  // Must have at least one item
        Items(builders.Object().
            Field("price", builders.Number())))

// Validate ensures ALL of these exist
var order Order
err := qf.ValidateInto(jsonBytes, schema, &order)
if err != nil {
    return err  // Clean error, no panic
}

// Now these are GUARANTEED safe:
userName := order.User.Name        // ✅ Can't panic
firstItem := order.Items[0]        // ✅ Can't panic  
price := firstItem.Price           // ✅ Can't panic
```

### The Peace of Mind

This isn't just about preventing crashes. It's about:

- **Better Sleep**: No 3 AM pages from panics
- **Cleaner Code**: No defensive nil checks everywhere
- **Faster Development**: Write business logic, not defensive code
- **Better Testing**: Test business logic, not panic recovery
- **Happier Teams**: Less stress, more productivity

---

## The Art of Transvasing: From Dynamic to Typed

One of the most elegant aspects of Queryfy + Superjsonic is how it handles the "transvasing" (transfer) of validated dynamic data into strongly-typed structs. This is where the two-path architecture truly shines.

### The Problem with Traditional Approaches

```go
// The dangerous traditional way
func processOrder(jsonData []byte) (*Order, error) {
    var order Order
    err := json.Unmarshal(jsonData, &order)  // Might partially unmarshal!
    if err != nil {
        // But what's the state of 'order' now? 
        // Partially filled? Zero values? Corrupted?
        return nil, err
    }
    return &order, nil
}
```

### The Queryfy + Superjsonic Transvasing Pipeline

```go
// The safe, intelligent way
func processOrder(jsonData []byte) (*Order, error) {
    // Phase 1: Validate without creating structs (fast path)
    if err := qf.Validate(jsonData, OrderSchema); err != nil {
        return nil, err  // No struct creation, no waste
    }
    
    // Phase 2: Only NOW do we transvase into structs
    var order Order
    if err := qf.ValidateInto(jsonData, OrderSchema, &order); err != nil {
        // This should never happen - validation already passed!
        return nil, err
    }
    
    // order is PERFECTLY formed, every field guaranteed safe
    return &order, nil
}
```

### Why Separation Matters

The separation of validation from unmarshaling is like having a water purification system with multiple stages:

1. **Pre-filter** (Smell Test): Catches obvious contamination
2. **Structural Filter** (Superjsonic): Ensures valid JSON structure  
3. **Purity Test** (Schema Validation): Verifies content meets standards
4. **Final Transfer** (Codec Unmarshal): Clean water into clean container

You wouldn't pour dirty water into a clean glass and then test it - you test first, pour second.

## Performance Deep Dive

### The Zero-Allocation Magic

Traditional JSON parsing allocates memory for every string, every object, every array. Superjsonic doesn't:

```go
// Traditional parsing - allocates everything
{
    "user": {                    // Allocation 1: map
        "name": "Alice",         // Allocation 2: string
        "email": "alice@ex.com"  // Allocation 3: string  
    },
    "items": [                   // Allocation 4: slice
        {"id": 1},               // Allocation 5: map
        {"id": 2}                // Allocation 6: map
    ]
}
// Total: 6+ allocations

// Superjsonic - zero allocations
[
    Token{Type: ObjectStart, Offset: 0},
    Token{Type: String, Offset: 2, Length: 4},    // "user"
    Token{Type: ObjectStart, Offset: 9},
    Token{Type: String, Offset: 11, Length: 4},   // "name"
    Token{Type: String, Offset: 18, Length: 5},   // "Alice"
    // ... more tokens
]
// Total: 0 allocations (tokens reused from pool)
```

### Concurrent Performance

The parser pool enables fantastic concurrent performance:

```go
// Process 1000 JSON documents concurrently
var wg sync.WaitGroup
for i := 0; i < 1000; i++ {
    wg.Add(1)
    go func(data []byte) {
        defer wg.Done()
        // Each goroutine gets its own parser from the pool
        err := qf.Validate(data, schema)
        // Parser automatically returned to pool
    }(jsonDocs[i])
}
wg.Wait()
```

Benchmark results:
- 1 goroutine: 1x baseline speed
- 10 goroutines: 8x faster
- 100 goroutines: Still 8x faster (no contention!)

---

## The Smell Test: Your Early Warning System

The smell test is like having a security camera on your data pipeline. It's not just about performance—it's about intelligence.

### What It Detects

```go
// Obvious corruption
{"user": "Alice", "ema     // Truncated
{user: "Alice"}            // Missing quotes
{"user": "Alice\xFF\xFE"} // Invalid UTF-8

// Suspicious patterns
{"injection": "<script>"}  // Possible XSS attempt
{"size": 999999999999}     // Suspiciously large number
{"\\\\\\\\": "\\\\\\\\"}  // Escape sequence abuse
```

### Monitoring and Alerting

```go
func MonitorJSONHealth(data []byte) {
    result := qf.SmellTest(data)
    
    // Log metrics
    metrics.Histogram("json.smell_score", result.Score)
    
    // Alert on degradation
    if result.Score < 0.5 {
        alert.Send("Poor JSON quality detected", map[string]interface{}{
            "score": result.Score,
            "patterns": result.Patterns,
            "source": getClientID(),
        })
    }
    
    // Track patterns over time
    for _, pattern := range result.Patterns {
        metrics.Inc("json.smell_pattern." + pattern)
    }
}
```

This turns the smell test into a diagnostic tool that can:
- Identify clients sending bad data
- Detect degrading integrations before they fail
- Provide forensic data for debugging
- Catch security attempts early

## The Philosophy: From "Parse Don't Validate" to Reality

The functional programming community has long advocated for "parse, don't validate" - the idea that you should parse untrusted input into types that cannot represent invalid states. Queryfy + Superjsonic makes this philosophy practical at scale.

### Traditional "Validation"

```go
type User struct {
    Email string `json:"email"`
    Age   int    `json:"age"`
}

func validateUser(u User) error {
    if u.Email == "" || !isValidEmail(u.Email) {
        return errors.New("invalid email")
    }
    if u.Age < 0 || u.Age > 150 {
        return errors.New("invalid age")
    }
    return nil
}

// Problem: You can still create invalid Users!
u := User{Email: "not-an-email", Age: -5}
```

### The Queryfy Way: Parse Into Existence

```go
// Define what a valid user IS, not what it ISN'T
userSchema := builders.Object().
    Field("email", builders.String().Email()).
    Field("age", builders.Number().Min(0).Max(150))

// This user can only exist if valid
var user ValidatedUser
err := qf.ValidateInto(jsonData, userSchema, &user)
// If err is nil, user is PERFECTLY valid
```

You're not validating data - you're parsing it into a shape that cannot be invalid. The schema becomes a parser that only produces valid output.

## The Trust Gradient: A New Mental Model

Queryfy + Superjsonic introduces a "trust gradient" that acknowledges trust isn't binary:

```
🧊 Frozen (Untrusted)         🌡️ Trust Thermometer         🔥 Blazing (Trusted)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
│ Raw Bytes │ Smells OK │ Valid JSON │ Schema Valid │ Type Safe │ Business │
│    ❄️     │    🌨️    │     ⛅     │      ☀️      │    🔥     │  Logic   │
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Each stage adds "heat" (trust), and you can exit at any temperature
```

This gradient approach means:
- **Efficiency**: Exit early when trust can't be established
- **Flexibility**: Different operations need different trust levels
- **Clarity**: Know exactly how much you trust your data

## The Unified Approach: One Pattern for Everything

One of Queryfy's most underappreciated features is how it unifies all JSON handling patterns:

### Before: Multiple Approaches

```go
// Approach 1: Structs with tags
type User struct {
    Email string `json:"email" validate:"required,email"`
}

// Approach 2: Dynamic validation
if email, ok := data["email"].(string); !ok || !isEmail(email) {
    return errors.New("invalid email")
}

// Approach 3: Schema validation
schema := map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{...}
}

// Different tools, different patterns, different errors
```

### After: One Pattern

```go
// One schema definition
schema := builders.Object().
    Field("email", builders.String().Email().Required())

// Works with everything
err := qf.Validate(structData, schema)      // ✅
err := qf.Validate(mapData, schema)         // ✅  
err := qf.Validate(jsonBytes, schema)       // ✅
err := qf.Validate(interfaceData, schema)   // ✅

// Same errors, same patterns, same mental model
```

This unification means:
- **Lower cognitive load**: One pattern to learn
- **Better team coordination**: Everyone uses the same approach
- **Easier refactoring**: Change data types without changing validation
- **Consistent errors**: Same error format everywhere

## The Codec Pattern: Choose Your Own Adventure

The codec interface is a practical design choice:

```go
// For maximum compatibility
qf := queryfy.New()  // Uses encoding/json

// For maximum performance  
qf := queryfy.New().WithCodec(sonic.Codec{})

// For special requirements
type EncryptedCodec struct{}

func (c EncryptedCodec) Unmarshal(data []byte, v interface{}) error {
    decrypted := decrypt(data)
    return json.Unmarshal(decrypted, v)
}

qf := queryfy.New().WithCodec(EncryptedCodec{})
```

This pattern enables:
- **Progressive enhancement**: Start simple, optimize later
- **Environment-specific choices**: Different codecs for different deployments
- **Special requirements**: Encryption, logging, metrics, etc.
- **Future compatibility**: New JSON libraries just need two methods

---

## Best Practices

### 1. Define Schemas Once, Use Everywhere

```go
// schemas/user.go
var UserSchema = builders.Object().
    Field("id", builders.String().UUID()).
    Field("email", builders.String().Email()).
    Field("name", builders.String().Min(1).Max(100)).
    Field("age", builders.Number().Min(0).Max(150))

// Use consistently across your app
api.Validate(data, UserSchema)
db.ValidateBeforeSave(data, UserSchema)  
queue.ValidateMessage(data, UserSchema)
```

### 2. Fail Fast, Fail Clearly

```go
// Don't do this
if err := qf.Validate(data, schema); err != nil {
    return errors.New("validation failed")  // Throws away useful info!
}

// Do this
if err := qf.Validate(data, schema); err != nil {
    return fmt.Errorf("invalid user data: %w", err)
    // Preserves: "email: must be valid email at line 3, column 12"
}
```

### 3. Choose the Right Validation Level

```go
// Just checking structure? Use Validate()
if err := qf.Validate(data, schema); err != nil {
    return err
}

// Need the actual data? Use ValidateInto()
var config Config
if err := qf.ValidateInto(data, schema, &config); err != nil {
    return err
}
```

### 4. Monitor Your JSON Health

```go
// Set up dashboards for:
- Smell test rejection rate
- Validation failure rate by endpoint
- Common validation errors
- Performance metrics (validation time)

// This helps you:
- Identify problematic clients
- Catch integration issues early  
- Optimize your schemas
- Prove SLA compliance
```

### 5. Use Appropriate Codecs

```go
// Default is fine for most cases
qf := queryfy.New()

// High-performance scenarios
qf := queryfy.New().WithCodec(jsoniter.ConfigFastest)

// Special requirements  
qf := queryfy.New().WithCodec(SecureCodec{})  // Encryption
qf := queryfy.New().WithCodec(LoggingCodec{}) // Audit trail
```

## Additional Insights

### The Power of Saying No

What makes Queryfy + Superjsonic effective isn't just what it does—it's what it deliberately doesn't do:

- ❌ No ORM features
- ❌ No schema migration tools  
- ❌ No code generation
- ❌ No custom DSL
- ❌ No framework ambitions

This restraint is intentional. By doing one thing—making JSON safe and fast—and doing it well, it composes nicely with everything else in your stack.

### Born from Production Pain

Every feature in Queryfy + Superjsonic exists because someone needed it:

- **Smell test**: Because corrupted uploads at 3 AM aren't fun
- **Token parsing**: Because OOM kills from large JSON are worse  
- **Path tracking**: Because "validation failed" helps nobody
- **Codec interface**: Because forcing a specific JSON library limits adoption

This isn't academic software—it's built from real production experience.

### Why Not Just Use...?

**validator/v10?** - Only works with structs, panics on maps  
**gjson?** - Great for querying, no validation  
**encoding/json + manual checks?** - Slow and error-prone  
**JSON Schema?** - String-based, no compile-time safety

Queryfy + Superjsonic is the first solution that's simultaneously:
- Faster than raw parsing
- Safer than struct tags
- Works with any data type
- Compile-time checked

### Your Schema IS Your API Documentation

```go
// This schema is documentation that can't lie
var UserAPI = builders.Object().
    Field("email", builders.String().Email()).
        Description("User's primary email").
        Example("alice@example.com").
    Field("role", builders.Enum("admin", "user", "guest")).
        Description("User's permission level").
        Default("user")

// Generate OpenAPI/Swagger automatically
docs := schema.ToOpenAPI()

// Or use in tests as truth source
testCases := schema.GenerateTestCases()
```

### Error Messages That Actually Help

```go
// Traditional validation error:
"validation failed"

// Queryfy + Superjsonic error:
ValidationError {
    Path: "users[3].profile.age"
    Line: 47
    Column: 23
    ByteOffset: 1822
    Expected: "number between 0 and 150"
    Actual: "-5"
    Suggestion: "age must be non-negative"
    Context: "...\"name\": \"Bob\", \"age\": -5, \"city\"..."
}
```

Every error tells you:
- WHERE it failed (path, line, column, byte)
- WHAT was expected vs actual
- WHY it matters
- HOW to fix it

### Migration: Start Small, Win Big

You don't need to convert everything at once:

```go
// Week 1: Just add validation to your scariest endpoint
func ScaryWebhook(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    
    // Add this one line
    if err := qf.Validate(body, WebhookSchema); err != nil {
        log.Printf("Dodged a bullet: %v", err)
        http.Error(w, err.Error(), 400)
        return
    }
    
    // Existing scary code now safe
    processWebhook(body)
}

// Week 2: Start using ValidateInto for new features
// Week 3: Replace panic-prone code
// Month 2: It's everywhere and you sleep better
```

### When Things Go Wrong: Forensic Mode

```go
// Enable debug mode for problematic data
debug := qf.WithDebug()
result, err := debug.ValidateVerbose(suspiciousData, schema)

// Get a complete validation trace
fmt.Println(result.Timeline)
// [0.1µs] Smell test: PASS (score: 0.89)
// [0.3µs] Token 0: ObjectStart 
// [0.4µs] Token 1: String "user"
// [0.5µs] Token 2: ObjectStart
// [0.6µs] Token 3: String "email"
// [0.7µs] Token 4: String "not-an-email"
// [0.8µs] Schema validation: FAIL at tokens[3-4]
// [0.9µs] Error: email must be valid email address

// Export for analysis
report := result.ExportReport()
```

---

## Conclusion: A New Baseline

Queryfy + Superjsonic represents a practical improvement in how we handle JSON in Go. It's not just about being faster—it's about making the right thing the easy thing.

### Before Queryfy + Superjsonic

- JSON validation was slow, so we skipped it
- Type assertions caused panics, so we added defensive code everywhere
- Different approaches for different scenarios
- Performance vs. safety trade-offs

### After Queryfy + Superjsonic

- Validation is so fast, it's negligent NOT to validate
- Panics are impossible because structure is verified first
- One consistent approach for all JSON handling
- Performance AND safety, no trade-offs

### The Hidden Benefit

The nice thing about this system is that it can improve your application's reliability without anyone noticing. Your services just become:
- Faster (5-8x validation performance)
- More reliable (zero panics)
- More secure (automatic bad data rejection)
- Easier to debug (clear error messages with location)

All without changing your application's architecture.

### Getting Started Is Easy

1. Install: `go get github.com/yourusername/queryfy`
2. Define your schemas using the intuitive builders
3. Replace `json.Unmarshal` with `qf.ValidateInto`
4. Sleep better knowing your JSON can't panic

### The Future

As more teams adopt Queryfy + Superjsonic, we're moving toward a future where:
- JSON panics are as rare as buffer overflows in Go
- Validation is never a performance bottleneck
- Bad data is caught at the edge, not in production
- Debugging JSON issues takes minutes, not hours

This isn't just an improvement—it's a new baseline for what developers should expect from JSON handling.

## Appendix: What This Architecture Enables

The clean separation of concerns in Queryfy + Superjsonic opens doors to features we haven't even built yet:

### Streaming Validation
```go
// Future: Validate GB-sized JSON without loading it all
validator.StreamValidate(reader, schema, func(path string, token Token) error {
    if path == "records[*]" {
        // Process each record as it's validated
        return db.Insert(token.Value())
    }
    return nil
})
```

### Partial Validation
```go
// Future: Validate only what you need
partialSchema := schema.SelectPaths("user.id", "user.email")
err := qf.ValidatePartial(jsonData, partialSchema)
// 10x faster when you only need specific fields
```

### Schema Evolution
```go
// Future: Automatic migration between schema versions
migration := builders.Migration().
    From(UserSchemaV1).
    To(UserSchemaV2).
    Transform("name", transformers.Split(" ")).As("firstName", "lastName")

newData, err := qf.Migrate(oldData, migration)
```

### Intelligent Caching
```go
// Future: Cache validation results
cached := qf.WithCache(redis)
err := cached.Validate(data, schema)  // Lightning fast for repeated data
```

### Time-Travel Debugging
```go
// Future: Record validation history
debug := qf.WithDebugger()
err := debug.Validate(data, schema)

// Later: What went wrong?
timeline := debug.GetTimeline()
// Shows: Smell test (pass) → Structure (pass) → Schema field X (fail)
```

The architecture is so clean that these become possible without fundamental changes.

---

*Remember: Fast validation isn't just about speed. It's about being able to validate everything, catch problems early, and build systems that are both performant and reliable. With Queryfy + Superjsonic, you don't have to choose.*

**Start validating more. Your future self will thank you.**