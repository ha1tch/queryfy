// formats.go - Custom format registry for string validation
package builders

import (
	"fmt"
	"sync"

	"github.com/ha1tch/queryfy"
)

// formatRegistry stores registered string format validators.
var formatRegistry = &formatStore{
	formats: make(map[string]queryfy.ValidatorFunc),
}

// formatStore is a thread-safe registry of named format validators.
type formatStore struct {
	mu      sync.RWMutex
	formats map[string]queryfy.ValidatorFunc
}

// RegisterFormat adds a named format with its validation function.
// Registered formats can be used via the FormatString() builder method
// on StringSchema.
//
// Formats are typically registered at init time:
//
//	func init() {
//	    builders.RegisterFormat("semver", func(value interface{}) error {
//	        // validate semver...
//	    })
//	}
//
// Calling RegisterFormat with a name that already exists overwrites
// the previous validator.
func RegisterFormat(name string, validator queryfy.ValidatorFunc) {
	formatRegistry.mu.Lock()
	defer formatRegistry.mu.Unlock()
	formatRegistry.formats[name] = validator
}

// LookupFormat returns the validator for a registered format, or nil
// if the format is not registered.
func LookupFormat(name string) queryfy.ValidatorFunc {
	formatRegistry.mu.RLock()
	defer formatRegistry.mu.RUnlock()
	return formatRegistry.formats[name]
}

// RegisteredFormats returns the names of all registered formats.
func RegisteredFormats() []string {
	formatRegistry.mu.RLock()
	defer formatRegistry.mu.RUnlock()
	names := make([]string, 0, len(formatRegistry.formats))
	for name := range formatRegistry.formats {
		names = append(names, name)
	}
	return names
}

// FormatString sets a registered format on the StringSchema.
// The format validator is looked up from the registry at schema
// construction time. If the format is not registered, validation
// will produce an error.
//
// This is the extensible alternative to the built-in Email(), URL(),
// and UUID() methods.
func (s *StringSchema) FormatString(name string) *StringSchema {
	s.formatType = name
	validator := LookupFormat(name)
	if validator != nil {
		s.validators = append(s.validators, validator)
	} else {
		// Store a validator that reports the missing format at validation time,
		// not at construction time. This allows formats to be registered after
		// schema construction (e.g., in tests).
		s.validators = append(s.validators, func(value interface{}) error {
			if LookupFormat(name) != nil {
				return LookupFormat(name)(value)
			}
			return fmt.Errorf("unknown format: %q", name)
		})
	}
	return s
}
