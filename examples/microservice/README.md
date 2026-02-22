# Microservice Examples

These examples demonstrate queryfy in realistic HTTP microservice contexts with external dependencies (routing, UUID generation, password hashing) that are **not** required by the core queryfy library.

## Examples

- **us/** — US-locale user registration and profile management API
- **ar/** — Argentine-locale variant with CUIT validation and Rioplatense formatting

## Building

These examples have their own `go.mod` with dependencies separate from the core library:

```bash
cd examples/microservice
go mod tidy
go run ./us
# or
go run ./ar
```

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/gorilla/mux` | HTTP routing |
| `github.com/google/uuid` | User ID generation |
| `golang.org/x/crypto` | Password hashing (bcrypt) |

These are **not** dependencies of queryfy itself.
