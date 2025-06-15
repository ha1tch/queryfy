# Superjsonic Architecture Evolution: From Insights to Implementation

## Introduction

This document captures how our architectural design evolved based on the key insights from our development session. It shows the transformation from a traditional validation architecture to an adaptive, probabilistic system that handles real-world JSON processing challenges.

---

# The Original Architecture (What We Started With)

## Traditional Validation Pipeline

```
JSON Input → Parse (encoding/json) → Unmarshal → Validate Schema → Result
              ↓                        ↓            ↓
           (Can fail)              (Can panic!)  (Too late!)
```

### Problems Discovered:
1. **All-or-nothing processing** - Spend full cost even for garbage
2. **Panic vulnerability** - Type mismatches cause runtime panics
3. **Late failure** - Errors discovered after expensive operations
4. **No corruption detection** - Binary/compressed data processed as JSON
5. **Memory inefficient** - Full object tree built before validation

---

# The Evolved Architecture

## Core Architectural Principles

Based on our insights, we established new principles:

1. **Fail Fast, Fail Cheap** - Reject bad data in microseconds
2. **Progressive Commitment** - Spend resources proportional to confidence
3. **Parallel Paths** - Validation separate from unmarshaling
4. **Adaptive Processing** - Learn from data patterns
5. **Diagnostic Capability** - When things fail, explain why

## The Multi-Stage Adaptive Pipeline

```
┌─────────────────────────────────────────────────────────────────┐
│                    INPUT: Raw JSON Bytes                         │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ STAGE 0: Intuition Layer (<1μs)                                 │
│ ┌─────────────┐ ┌──────────────┐ ┌────────────────┐           │
│ │ First Bytes │ │Entropy Check │ │ Magic Numbers  │           │
│ │   Analysis  │ │  (64 bytes)  │ │   Detection    │           │
│ └─────────────┘ └──────────────┘ └────────────────┘           │
│         ↓               ↓                ↓                      │
│         └───────────────┴────────────────┘                     │
│                         ↓                                       │
│                  Smell Quality                                  │
│                 (Fresh/Mild/Fishy/Rotten)                      │
└────────────────────────────┬────────────────────────────────────┘
                             │
                    ┌────────┴────────┐
                    │ Rotten?         │
                    │                 │
                    └───┬─────────┬───┘
                        │ Yes     │ No
                        ▼         ▼
                   [REJECT]    Continue
                   (<1μs)         │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│ STAGE 1: Structural Validation (<100μs)                         │
│ ┌─────────────────┐ ┌──────────────────┐ ┌─────────────────┐  │
│ │  Superjsonic    │ │ Token Generation │ │ Balance Check   │  │
│ │    Parser       │ │  (Zero Alloc)    │ │   Validation    │  │
│ └─────────────────┘ └──────────────────┘ └─────────────────┘  │
│                            ↓                                    │
│                      Token Stream                               │
└────────────────────────────┬────────────────────────────────────┘
                             │
                    ┌────────┴────────┐
                    │ Parse Error?    │
                    │                 │
                    └───┬─────────┬───┘
                        │ Yes     │ No
                        ▼         ▼
                [REJECT + Smellmap]  Continue
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────┐
│ STAGE 2: Type Validation (<1ms)                                 │
│ ┌─────────────────┐ ┌──────────────────┐ ┌─────────────────┐  │
│ │ Token Type      │ │ Schema Type      │ │ Path Tracking   │  │
│ │   Checking      │ │   Matching       │ │   & Errors      │  │
│ └─────────────────┘ └──────────────────┘ └─────────────────┘  │
│                            ↓                                    │
│                    Type Compatibility                           │
└────────────────────────────┬────────────────────────────────────┘
                             │
                    ┌────────┴────────┐
                    │ Type Mismatch?  │
                    │                 │
                    └───┬─────────┬───┘
                        │ Yes     │ No
                        ▼         ▼
                  [REJECT]    Continue
              (Prevents Panic!)      │
                                    ▼
┌─────────────────────────────────────────────────────────────────┐
│ STAGE 3: Schema Rules (<10ms)                                   │
│ ┌─────────────────┐ ┌──────────────────┐ ┌─────────────────┐  │
│ │ Field Rules     │ │ Dependencies     │ │ Custom Logic    │  │
│ │ (min/max/regex) │ │   Validation     │ │   Validation    │  │
│ └─────────────────┘ └──────────────────┘ └─────────────────┘  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ Validation Only?│
                    └───┬─────────┬───┘
                        │ Yes     │ No
                        ▼         ▼
                    [SUCCESS]  Unmarshal Path
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│ STAGE 4: Unmarshal (Optional)                                   │
│ ┌─────────────────┐ ┌──────────────────┐ ┌─────────────────┐  │
│ │ User's Codec    │ │ Safe Unmarshal   │ │ Final Result    │  │
│ │ (json/jsoniter) │ │ (No Panic!)      │ │                 │  │
│ └─────────────────┘ └──────────────────┘ └─────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Cost-Based Decision Architecture

### Computational Budget Allocation

```go
type ProcessingBudget struct {
    MaxTime        time.Duration
    MaxMemory      int64
    MaxComplexity  int
}

type AdaptiveProcessor struct {
    quickBudget    ProcessingBudget // <1μs, 0 allocs
    normalBudget   ProcessingBudget // <1ms, minimal allocs
    deepBudget     ProcessingBudget // <10ms, some allocs
    forensicBudget ProcessingBudget // <100ms, diagnostic allocs
}
```

### Decision Tree Implementation

```go
func (ap *AdaptiveProcessor) Process(data []byte) ProcessingResult {
    startTime := time.Now()
    
    // Level 0: Intuition (always run)
    smell := ap.quickSmell(data)
    if smell.Quality < 0.2 {
        return ProcessingResult{
            Decision: RejectImmediately,
            Cost:     time.Since(startTime),
            Stage:    "Intuition",
            Reason:   smell.Reason,
        }
    }
    
    // Adaptive budget based on smell
    budget := ap.selectBudget(smell.Quality)
    
    // Level 1: Structural (conditional)
    if budget.MaxTime > 100*time.Microsecond {
        structural := ap.structuralCheck(data, budget)
        if structural.Failed {
            if smell.Quality < 0.5 {
                // Low confidence + structural failure = generate smellmap
                return ap.generateDiagnostics(data, structural)
            }
            return ProcessingResult{
                Decision: RejectStructural,
                Cost:     time.Since(startTime),
                Stage:    "Structural",
                Errors:   structural.Errors,
            }
        }
    }
    
    // Continue with type validation, schema rules, etc.
    // Each stage checks budget before proceeding
}
```

## Parallel Path Architecture

### The Dual-Path Design

```
                    ┌──────────────────┐
                    │   User Request   │
                    └────────┬─────────┘
                             │
                ┌────────────┴────────────┐
                │     Path Decision       │
                │  (Based on API Call)    │
                └──┬──────────────────┬───┘
                   │                  │
        Validate() │                  │ ValidateInto()
                   ▼                  ▼
         ┌─────────────────┐  ┌─────────────────┐
         │   Fast Path     │  │   Full Path     │
         │                 │  │                 │
         │ 1. Intuition    │  │ 1. Intuition    │
         │ 2. Structure    │  │ 2. Structure    │
         │ 3. Types        │  │ 3. Types        │
         │ 4. Schema       │  │ 4. Schema       │
         │                 │  │ 5. Unmarshal    │
         │ [No Unmarshal]  │  │ 6. Type Safety  │
         └─────────────────┘  └─────────────────┘
                   │                  │
                   ▼                  ▼
              Validation         Object + Valid
              Result Only           Result
```

### Path Selection Logic

```go
// Fast Path - Most Common
func (q *Queryfy) Validate(data []byte, schema Schema) error {
    // Uses Superjsonic only
    // Zero allocations
    // Returns only error
}

// Full Path - When Needed
func (q *Queryfy) ValidateInto(data []byte, schema Schema, v interface{}) error {
    // First: Fast validation
    if err := q.Validate(data, schema); err != nil {
        return err
    }
    
    // Then: Safe unmarshal with user's codec
    return q.codec.Unmarshal(data, v)
}
```

## Diagnostic Architecture

### When Things Go Wrong

```
                 Validation Failure
                        │
         ┌──────────────┴──────────────┐
         │   Failure Classification    │
         └──────────────┬──────────────┘
                        │
    ┌───────────────────┼───────────────────┐
    │                   │                   │
    ▼                   ▼                   ▼
Corrupted           Structural          Business
  Data               Error               Rule
    │                   │                   │
    ▼                   ▼                   ▼
Generate           Token Path         Clear Error
Smellmap           Diagnostics          Message
    │                   │                   │
    ▼                   ▼                   ▼
┌─────────┐      ┌──────────┐      ┌──────────┐
│Heatmap  │      │Path:     │      │Expected: │
│▓▓▓░░░░░│      │users[0]. │      │ >0       │
│████▓░░░│      │ email    │      │Got: -5   │
└─────────┘      └──────────┘      └──────────┘
```

### Smellmap Integration

```go
type DiagnosticPipeline struct {
    threshold float64 // When to generate diagnostics
}

func (dp *DiagnosticPipeline) ShouldDiagnose(
    smell SmellQuality, 
    error error,
) bool {
    // Generate diagnostics for:
    // 1. Suspicious data that failed
    // 2. Clean-looking data that failed (unexpected)
    // 3. User-requested diagnostics
    
    return smell == SmellFishy || 
           (smell == SmellMild && error != nil) ||
           dp.diagnosticsEnabled
}
```

## Learning and Adaptation Layer

### Runtime Optimization

```go
type AdaptivePipeline struct {
    stats     RuntimeStats
    patterns  PatternCache
    thresholds DynamicThresholds
}

type RuntimeStats struct {
    avgCleanProcessTime   time.Duration
    avgCorruptRejectTime  time.Duration
    corruptionRate        float64
    commonFailurePatterns []Pattern
}

func (ap *AdaptivePipeline) Adapt() {
    // Adjust thresholds based on recent data
    if ap.stats.corruptionRate > 0.5 {
        // High corruption environment
        ap.thresholds.intuitionCutoff = 0.7  // More aggressive
        ap.thresholds.enableSmellmap = true  // Auto-diagnose
    } else {
        // Clean environment
        ap.thresholds.intuitionCutoff = 0.3  // More permissive
        ap.thresholds.enableSmellmap = false // Save resources
    }
}
```

## Integration Points

### How Components Connect

```
┌──────────────────────────────────────────────────────┐
│                    Queryfy Core                       │
│  ┌──────────────┐  ┌────────────┐  ┌─────────────┐ │
│  │   Schemas    │  │   Codec    │  │   Errors    │ │
│  │  (Builders)  │  │ Interface  │  │  (Detailed) │ │
│  └──────┬───────┘  └─────┬──────┘  └──────┬──────┘ │
│         │                 │                 │        │
│         └─────────────────┼─────────────────┘        │
│                           ▼                          │
│                  ┌─────────────────┐                 │
│                  │ Pipeline Engine │                 │
│                  └────────┬────────┘                 │
└──────────────────────────┼──────────────────────────┘
                           │
          ┌────────────────┼────────────────┐
          │                │                │
          ▼                ▼                ▼
   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
   │  Intuition  │  │ Superjsonic │  │ User Codec  │
   │   System    │  │   Parser    │  │  (External) │
   └─────────────┘  └─────────────┘  └─────────────┘
```

### Module Boundaries

1. **Intuition System**: Probabilistic pre-filter
2. **Superjsonic**: Zero-allocation tokenizer
3. **Pipeline Engine**: Orchestrates stages
4. **Schema System**: Validation rules
5. **Codec Interface**: User's JSON library
6. **Diagnostic System**: Smellmaps and forensics

## Performance Characteristics

### Expected Performance by Path

```
Path                Input Size    Time        Allocations
----                ----------    ----        -----------
Intuition Only      Any          <1μs        0
Corrupt Reject      Any          <10μs       0
Small Valid JSON    <1KB         <100μs      0
Large Array Valid   >100KB       <10ms       1 (token array)
With Unmarshal      Any          +codec time +codec allocs
With Diagnostics    Any          +100μs      +smellmap
```

### Concurrency Model

```go
// Parser pooling for zero-allocation under load
var parserPool = sync.Pool{
    New: func() interface{} {
        return &CompositeParser{
            intuition:   NewIntuitionChecker(),
            structural:  NewSuperjsonicParser(),
            type:        NewTypeValidator(),
            schema:      NewSchemaValidator(),
        }
    },
}

// Each goroutine gets its own parser
func HandleConcurrentRequest(data []byte) error {
    parser := parserPool.Get().(*CompositeParser)
    defer parserPool.Put(parser)
    
    return parser.Process(data)
}
```

## Error Flow Architecture

### Progressive Error Detail

```
Level 0 (Intuition):    "Invalid JSON structure"
                              ↓
Level 1 (Structural):   "Unexpected character '}' at byte 234"
                              ↓
Level 2 (Type):         "users[0].age: expected number, got string"
                              ↓
Level 3 (Schema):       "users[0].age: must be between 0 and 150, got 200"
                              ↓
Level 4 (Business):     "Order total doesn't match sum of items"
```

### Error Aggregation

```go
type ErrorCollector struct {
    errors    []ValidationError
    maxErrors int
    strategy  ErrorStrategy
}

type ErrorStrategy int

const (
    CollectAll    ErrorStrategy = iota // Get all errors
    FailFast                            // Stop on first
    FailPerPath                         // One per JSON path
    FailPerType                         // One per error type
)
```

## Deployment Architecture

### Configuration Modes

```go
type DeploymentMode string

const (
    // Maximum safety, some performance cost
    ModeProduction DeploymentMode = "production"
    
    // Balance of safety and speed
    ModeBalanced DeploymentMode = "balanced"
    
    // Maximum performance, trust input more
    ModePerformance DeploymentMode = "performance"
    
    // Full diagnostics, testing
    ModeDevelopment DeploymentMode = "development"
)

func (q *Queryfy) SetMode(mode DeploymentMode) {
    switch mode {
    case ModeProduction:
        q.pipeline.EnableAllStages()
        q.pipeline.SetThresholds(Conservative)
    case ModePerformance:
        q.pipeline.DisableStage(StageIntuition)
        q.pipeline.SetThresholds(Aggressive)
    // etc...
    }
}
```

## Future Architecture Extensions

### Stream Processing Integration

```
Stream Input → Chunk Buffer → Intuition → Parse Chunk → Validate → Emit
     ↑                                                                ↓
     └────────────────── Continue with next chunk ←──────────────────┘
```

### Distributed Validation

```
                Load Balancer
                     │
        ┌────────────┼────────────┐
        ▼            ▼            ▼
    Worker 1     Worker 2     Worker 3
    (Intuition)  (Structure)  (Schema)
        │            │            │
        └────────────┼────────────┘
                     ▼
                Aggregator
```

---

## Conclusion: Architecture as Response to Reality

This architecture evolved from our discoveries:

1. **Intuition Layer** - Because most corruption is obvious
2. **Progressive Pipeline** - Because resources should match confidence  
3. **Dual Paths** - Because validation ≠ unmarshaling
4. **Diagnostic System** - Because failures need explanation
5. **Adaptive Processing** - Because patterns emerge at runtime

The architecture isn't elegant for its own sake—every component exists to solve a real problem we discovered during development.

---

*This architecture represents lessons learned through experimentation and failure. It's optimized for the real world, not theoretical purity.*