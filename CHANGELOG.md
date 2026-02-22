# Changelog

## v0.3.0 — 2026-02-22

Schema introspection, JSON Schema interoperability, wildcard queries,
iteration methods, async validation, and schema compilation. Zero
external dependencies. 499 tests, 80%+ coverage across all packages.

### Added

**Schema introspection and tooling**

- Schema introspection API: `GetField`, `FieldNames`, `RequiredFieldNames`,
  `LengthConstraints`, `RangeConstraints`, `MultipleOfValue`, `IsInteger`,
  `EnumValues`, `FormatType`, `PatternString`, `ElementSchema`,
  `ItemCountConstraints`, `IsUniqueItems`, `AllowsAdditional` — full read
  access to every constraint on every schema type.
- Schema equality (`builders.Equal`) and canonical hashing
  (`builders.Hash`) for structural comparison of schemas.
- Schema diff (`builders.Diff`) returns a list of field-level changes
  (added, removed, modified) between two object schemas.
- Field walker (`builders.Walk`) with visitor callbacks for recursive
  schema traversal.
- Custom format registry (`builders.RegisterFormat`,
  `builders.LookupFormat`) for user-defined string format validators.
- Type metadata (`Meta`, `GetMeta`, `AllMeta`) on all schema types for
  storing arbitrary key-value pairs alongside validation rules.

**JSON Schema interoperability**

- `builders/jsonschema` sub-package with `FromJSON` (import) and `ToJSON`
  (export).
- Supports a practical subset of JSON Schema Draft 2020-12 / Draft 7:
  all primitive types, string constraints (minLength, maxLength, pattern,
  enum, format), number constraints (minimum, maximum, exclusiveMinimum,
  exclusiveMaximum, multipleOf), object keywords (properties, required,
  additionalProperties), array keywords (items, minItems, maxItems,
  uniqueItems), nullable (both `nullable: true` and `type: ["string", "null"]`),
  type inference from properties/items.
- Unsupported features (`$ref`, `oneOf`/`anyOf`/`allOf`/`not`,
  `if`/`then`/`else`, `patternProperties`, `prefixItems`, and others) produce
  clear errors with document paths rather than silent data loss.
- `Options.StrictMode` controls whether unsupported features are errors or
  warnings. `Options.StoreUnknown` preserves unrecognised keywords (including
  `x-` extensions) as schema metadata.
- `ExportOptions.SchemaURI` and `ExportOptions.ID` set `$schema` and `$id`.
  `ExportOptions.IncludeMeta` emits stored metadata as extension keywords.
- Round-trip fidelity: import a JSON Schema document, export it, re-import
  the output, and verify structural equality. 9 round-trip tests cover
  strings, numbers, integers, booleans, objects, arrays, nullable types,
  nested structures, and custom extensions.
- Full documentation: `doc/jsonschema-interop-EN.md` and runnable example
  `examples/jsonschema/main.go`.

**Query and iteration**

- Wildcard queries: `items[*].price` returns all prices,
  `customers[*].orders[*].total` flattens nested arrays. Wildcards work in
  both `query.Execute` and `query.ExecuteCached`.
- `queryfy.Each(data, path, fn)` iterates over wildcard-matched elements
  with early-stop support.
- `queryfy.Collect(data, path, fn)` applies a transform to each matched
  element and returns the results.
- `queryfy.ValidateEach(data, path, schema, mode)` validates each matched
  element against a schema, producing indexed error paths
  (e.g., `[2].name: field is required`).

**Async validation**

- `AsyncValidatorFunc` type for validators that need I/O (database lookups,
  API calls, etc.).
- `AsyncCustom` method on StringSchema, NumberSchema, BoolSchema,
  ObjectSchema, and ArraySchema.
- `ValidateAndTransformAsync` on TransformSchema, ObjectSchema, and
  ArraySchema — runs sync validation first, then async validators
  sequentially with context propagation.
- Cancellation via `context.Context` propagates through all async paths.

**Schema compilation**

- `queryfy.Compile(schema)` pre-resolves all constraint checks into a flat
  `[]checkFunc` slice. At validation time, the compiled schema executes one
  loop with no nil-checks or conditional branching on which constraints are
  configured.
- Compilation is recursive: compiled object schemas contain compiled field
  schemas.
- Enum lookups are pre-built as `map[string]bool` for O(1) membership
  testing.
- Required-field checks are pre-merged from both `RequiredFields()` and
  individual field `Required()` calls into a single boolean per field.
- `Compile` is idempotent: compiling an already-compiled schema returns it
  unchanged.
- Unsupported schema types (composite, dependent, datetime, custom) fall back
  to delegating to the original `Validate` method.

**Infrastructure**

- `AllowAdditional(bool)` on ObjectSchema, decoupled from validation mode.
  When set explicitly, overrides the mode-based default. When unset, strict
  mode rejects extra fields and loose mode allows them (unchanged behaviour).
- `AllowsAdditional()` getter returns `(allow bool, explicit bool)`.
- `Validators()` getter on all schema types exposes custom validator
  functions (used by the compiler).
- `PatternMatch(string)` on StringSchema exposes the compiled regex for
  external use.
- `IsUniqueItems()` getter on ArraySchema.
- GitHub Actions CI: test matrix across Go versions, golangci-lint, example
  builds.
- `.golangci.yml` configuration.
- Makefile `PACKAGES` variable updated to include `builders/jsonschema/`.
- `ValidationContext.Reset()` clears accumulated errors, path state, and
  transformation records for context reuse without reallocation.
- `compile_bench_test.go` benchmark suite comparing raw vs compiled schema
  validation across all schema types.

### Changed

- `Compile()` is no longer a no-op. It now returns a `*CompiledSchema` with
  pre-resolved validation logic. Code that called `Compile` and expected the
  original schema type back will now receive a `*CompiledSchema`. The
  `Inner()` method provides access to the original.
- v0.2.0 is now labelled "Released" in the roadmap (was "Current Release").
- Roadmap restructured: original v0.3.0 plan items (wildcard queries,
  iteration, compilation) completed; loose-mode data transformation deferred
  to v0.4.0; JSON Schema compatibility promoted from v0.4.0 to v0.3.0.

### Fixed

- DateTime `StrictFormat` was silently ignored when set before `Format`,
  causing non-strict parsing regardless of call order. Fixed to apply the
  strict flag at validation time.
- DependentSchema accepted empty field names in `WhenPresent`/`WhenAbsent`,
  leading to schemas that could never trigger. Now returns an error.
- `ValidateAndTransform` was missing on ObjectSchema and ArraySchema,
  causing the transform pipeline to skip field-level and element-level
  transformations. Implemented for both.

### Removed

- `internal/` package (unused).
- `validators/` package (unused).
- `WithTransform` method on ObjectSchema (dead code, replaced by the
  transform pipeline in v0.2.0).

### Metrics

| Metric | v0.2.0 (start) | v0.3.0 (end) |
|---|---|---|
| Tests | ~100 | 499 |
| Packages | 5 | 6 (`builders/jsonschema` added) |
| Implementation lines | ~4,500 | 7,693 |
| Test lines | ~2,500 | 9,238 |
| Coverage (root) | ~45% | 84% |
| Coverage (builders) | ~55% | 84% |
| Coverage (jsonschema) | — | 90% |
| Coverage (query) | ~40% | 72% |
| CI | None | GitHub Actions |
