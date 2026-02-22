# Queryfy
#### Validate, Query, and Transform Dynamic Data in Go

Queryfy is a Go library for working with `map[string]interface{}` data. It provides
schema validation, path-based querying, and data transformation in a single package —
designed for JSON APIs, configuration files, and any scenario where data arrives as
untyped maps rather than structs.

Zero external dependencies. 499 tests. 80%+ coverage.

## Features

**Validation:**
schema definitions via fluent builder API, strict and loose modes, composite logic
(AND/OR/NOT), custom validators, async validators with context cancellation, dependent
field validation, DateTime with format/range/business-day constraints, clear error
messages with exact field paths.

**Querying:**
dot-notation path expressions, array indexing, wildcard expansion (`items[*].price`),
iteration methods (`Each`, `Collect`, `ValidateEach`).

**Transformation:**
31 built-in transformers (string, number, date, phone), custom transformers, transform
chaining, validate-and-transform in a single pass.

**Schema tooling:**
introspection API (read constraints, walk fields, compare schemas), JSON Schema import
and export with round-trip verification, schema compilation for 25–53% faster validation,
custom format registry, type metadata.

## Installation

```bash
go get github.com/ha1tch/queryfy
```

Requires Go 1.22 or later.

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    qf "github.com/ha1tch/queryfy"
    "github.com/ha1tch/queryfy/builders"
)

func main() {
    // Define a schema
    schema := builders.Object().
        Field("customerId", builders.String().Required()).
        Field("amount", builders.Number().Min(0).Required()).
        Field("items", builders.Array().Of(
            builders.Object().
                Field("productId", builders.String().Required()).
                Field("quantity", builders.Number().Min(1)).
                Field("price", builders.Number().Min(0)),
        ).MinItems(1))

    // Data to validate
    order := map[string]interface{}{
        "customerId": "CUST-123",
        "amount":     150.50,
        "items": []interface{}{
            map[string]interface{}{
                "productId": "PROD-456",
                "quantity":  float64(2),
                "price":     75.25,
            },
        },
    }

    // Validate
    if err := qf.Validate(order, schema); err != nil {
        log.Fatal(err)
    }

    // Query
    price, _ := qf.Query(order, "items[0].price")
    fmt.Printf("First item price: $%.2f\n", price)
}
```

## Error Messages

Queryfy produces clear, actionable errors with exact paths:

```
validation failed:
  customer.email: must be a valid email address, got "not-an-email"
  items[0].quantity: must be >= 1, got 0
  items[2].productId: field is required
  payment.method: must be one of: CARD, CASH, DIGITAL_WALLET, got "CHECK"
```

## Documentation

**Usage guide:**
- **[MANUAL.md](MANUAL.md)** — Full API reference with examples for every feature

**Design and rationale:**
- **[Why Queryfy?](https://github.com/ha1tch/queryfy/blob/main/doc/queryfy-why-EN.md)** — The problem queryfy solves and why it matters
- **[Design Philosophy](https://github.com/ha1tch/queryfy/blob/main/doc/design-and-innovations_EN_v0.1.0_2025-06-13.md)** — Core design principles and innovations
- **[Comparison Guide](https://github.com/ha1tch/queryfy/blob/main/doc/COMPARISON.md)** — How queryfy compares to go-playground/validator, gjson, and others
- **[JSON Schema Interoperability](https://github.com/ha1tch/queryfy/blob/main/doc/jsonschema-interop-EN.md)** — Importing and exporting JSON Schema, round-tripping, supported features
- **[Schema-Struct Synchronization](https://github.com/ha1tch/queryfy/blob/main/doc/queryfy-schema-struct-sync-EN.md)** — Moving validated data into structs

**Philosophy:**
- **[Compose at Build-Time](https://github.com/ha1tch/queryfy/blob/main/doc/philosophy-en-01.md)** — The fundamental principle
- **[Practical Ramifications](https://github.com/ha1tch/queryfy/blob/main/doc/philosophy-en-02.md)** — Real-world implications and patterns

**En Español:**
- **[¿Por qué Queryfy?](https://github.com/ha1tch/queryfy/blob/main/doc/queryfy-why-ES.md)** — Entendiendo el problema que queryfy resuelve
- **[Sincronización Esquema-Struct](https://github.com/ha1tch/queryfy/blob/main/doc/queryfy-schema-struct-sync-ES.md)** — Datos validados en structs
- **[Filosofía de Diseño](https://github.com/ha1tch/queryfy/blob/main/doc/design-and-innovations_ES_v0.1.0_2025-06-13.md)** — Principios de diseño e innovaciones
- **[Componer en Tiempo de Compilación](https://github.com/ha1tch/queryfy/blob/main/doc/philosophy-es-01.md)** — El principio fundamental
- **[Ramificaciones Prácticas](https://github.com/ha1tch/queryfy/blob/main/doc/philosophy-es-02.md)** — Implicaciones y patrones del mundo real

**Research:**
- **[What Developers Need](https://github.com/ha1tch/queryfy/blob/main/doc/research-01-market-needs.md)** — Analysis of Go ecosystem validation needs
- **[How Queryfy Delivers](https://github.com/ha1tch/queryfy/blob/main/doc/research-02-how-qf-delivers.md)** — Mapping features to developer needs

## Performance

Queryfy schemas are defined once and reused. Query paths are cached after first use.
The library uses type-switch optimisation instead of reflection and has no external
dependencies.

For hot paths, `Compile()` pre-processes a schema into an optimised form that eliminates
per-validation overhead. The compiled schema is a drop-in replacement:

```go
var orderSchema = qf.Compile(builders.Object().
    Field("id", builders.String().Required()).
    Field("amount", builders.Number().Min(0)))
```

Measured with `go test -bench=. -benchmem -count=3` on Xeon Platinum 8581C:

| Benchmark | Raw | Compiled | Delta |
|-----------|-----|----------|-------|
| Object (valid) | 1,360 ns/op · 0 B · 0 allocs | 1,020 ns/op · 0 B · 0 allocs | -25% time |
| Object (invalid) | 3,000 ns/op · 427 B · 24 allocs | 2,500 ns/op · 395 B · 22 allocs | -17% time · -8% allocs |
| Number | 40 ns/op · 0 B · 0 allocs | 22 ns/op · 0 B · 0 allocs | -44% time |
| Enum (10 values) | 47 ns/op · 0 B · 0 allocs | 22 ns/op · 0 B · 0 allocs | -53% time |
| Array (20 items) | 2,080 ns/op · 96 B · 21 allocs | 2,140 ns/op · 96 B · 21 allocs | flat (element validation dominates) |
| String (email) | 395 ns/op · 0 B · 0 allocs | 411 ns/op · 0 B · 0 allocs | flat (regex dominates) |
| **Compile cost** | — | 4,520 ns/op · 2,840 B · 57 allocs | one-time; amortised after ~4 validations |

## Subprojects

**[Superjsonic](https://github.com/ha1tch/queryfy/tree/main/superjsonic#readme)** —
a fast JSON pre-validation parser that will be merged into queryfy in a future release.

## Roadmap

### v0.1.0 (Released)
- [✓] Schema validation with builder API
- [✓] Basic path queries (dot notation, array indexing)
- [✓] Composite schemas (AND/OR/NOT)
- [✓] Strict and loose validation modes
- [✓] Custom validators
- [✓] Clear error messages with paths

### v0.2.0 (Released)
- [✓] Data transformation pipeline with builder pattern
- [✓] DateTime validation with comprehensive format support
- [✓] Dependent field validation for conditional requirements
- [✓] Phone normalization for multiple countries
- [✓] Built-in transformers (string, number, date operations)
- [✓] Transform chaining with `.Add()` method

### v0.3.0 (Current Release)

v0.3.0 diverged from the original plan in response to real-world usage. Development
of a downstream application revealed that queryfy's schemas were opaque at runtime:
there was no way to inspect constraints, compare schemas, traverse fields
programmatically, or convert between queryfy and external schema formats. These
capabilities were prerequisites for building tooling on top of queryfy — schema-driven
form generation, API contract validation, migration tooling, or integration with
systems that already use JSON Schema.

The introspection API was built first, then JSON Schema import/export on top of it,
validated with round-trip tests. Once the introspection layer was in place, the
originally planned query features and schema compilation were straightforward additions.

**Schema introspection and tooling:**
- [✓] Schema introspection API (`GetField`, `RequiredFieldNames`, `RangeConstraints`, etc.)
- [✓] Schema equality and hashing (structural comparison, canonical hashing)
- [✓] Schema diff (field-level change detection between schema versions)
- [✓] Field walker (recursive traversal with visitor callbacks)
- [✓] Custom format registry (`RegisterFormat`, `LookupFormat`)
- [✓] Custom type metadata (`Meta`, `GetMeta`, `AllMeta` on all schema types)

**Interoperability:**
- [✓] JSON Schema import (`builders/jsonschema.FromJSON`) — Draft 2020-12 / Draft 7 subset
- [✓] JSON Schema export (`builders/jsonschema.ToJSON`) with round-trip verification
- [✓] Controllable additional properties (`AllowAdditional` decoupled from validation mode)

**Query and iteration:**
- [✓] Wildcard queries (`items[*].price`, nested `customers[*].orders[*].total`)
- [✓] Iteration methods (`Each`, `Collect`, `ValidateEach`)
- [✓] Enhanced transform API (`.Transform()` convenience methods on leaf schemas)

**Async validation:**
- [✓] Context-aware async validators (`AsyncCustom`, `ValidateAndTransformAsync`)
- [✓] Cancellation propagation through field and object-level async validators

**Infrastructure:**
- [✓] Schema compilation (`Compile` flattens constraint checks into a single function-call chain)
- [✓] GitHub Actions CI (test matrix, lint, example builds)
- [✓] golangci-lint configuration
- [✓] 80%+ test coverage across all packages

### v0.4.0 (Planned)

The items deferred to v0.4.0 require semantic changes to how queryfy processes data,
not just how it reads or describes schemas.

**Query layer enhancements:**
- [ ] Filter expressions (`items[?price > 100]`) — predicate evaluation inside brackets
- [ ] Aggregation functions (`sum()`, `avg()`, `count()`) — operates on wildcard results

**Data transformation:**
- [ ] Data transformation in loose mode (modify actual data, not just validate type
  compatibility) — requires careful design around coercion policy and failure reporting

### v0.5.0 (Future)
- [ ] Struct conversion (`ToStruct`, `ValidateToStruct`)
- [ ] Pre-validation of raw JSON bytes before unmarshalling
- [ ] Schema `Explain()` method for human-readable rule descriptions

## License

Copyright 2025-2026 h@ual.fi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

<http://www.apache.org/licenses/LICENSE-2.0>

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
