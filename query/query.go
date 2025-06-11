// Package query provides a simple query language for navigating data structures.
//
// The query language supports:
//   - Field access: "field" or "object.field"
//   - Array indexing: "array[0]" or "array[0].field"
//   - Nested access: "user.address.street"
//
// Example:
//
//	data := map[string]interface{}{
//		"user": map[string]interface{}{
//			"name": "John",
//			"emails": []interface{}{
//				"john@example.com",
//				"john.doe@company.com",
//			},
//		},
//	}
//
//	name, _ := query.Execute(data, "user.name")         // "John"
//	email, _ := query.Execute(data, "user.emails[0]")  // "john@example.com"
package query

import (
	"sync"
)

// QueryCache caches parsed queries for performance.
var queryCache = &cache{
	queries: make(map[string]*Query),
}

// cache is a simple thread-safe cache for parsed queries.
type cache struct {
	mu      sync.RWMutex
	queries map[string]*Query
}

// get retrieves a query from the cache.
func (c *cache) get(key string) (*Query, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	q, ok := c.queries[key]
	return q, ok
}

// set stores a query in the cache.
func (c *cache) set(key string, query *Query) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple cache eviction: if cache is too large, clear it
	if len(c.queries) > 1000 {
		c.queries = make(map[string]*Query)
	}

	c.queries[key] = query
}

// ExecuteCached executes a query with caching.
// This is the recommended way to execute queries in production.
func ExecuteCached(data interface{}, queryStr string) (interface{}, error) {
	// Check cache first
	if cached, ok := queryCache.get(queryStr); ok {
		path := SimplifyNode(cached.Root.Child)
		return ExecutePath(data, path)
	}

	// Parse and cache
	query, err := ParseQuery(queryStr)
	if err != nil {
		return nil, err
	}

	queryCache.set(queryStr, query)

	path := SimplifyNode(query.Root.Child)
	return ExecutePath(data, path)
}

// ClearCache clears the query cache.
// This is mainly useful for testing.
func ClearCache() {
	queryCache.mu.Lock()
	defer queryCache.mu.Unlock()
	queryCache.queries = make(map[string]*Query)
}
