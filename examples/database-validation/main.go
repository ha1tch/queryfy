package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "strings"
    "sync"
    "time"
    
    qf "github.com/ha1tch/queryfy"
    "github.com/ha1tch/queryfy/builders"
    "github.com/ha1tch/queryfy/builders/transformers"
)

// Mock database for demonstration
type MockDB struct {
    users     map[string]User
    usernames map[string]bool
    mu        sync.RWMutex
}

type User struct {
    ID        string
    Email     string
    Username  string
    CreatedAt time.Time
}

func NewMockDB() *MockDB {
    return &MockDB{
        users:     make(map[string]User),
        usernames: make(map[string]bool),
    }
}

func (db *MockDB) EmailExists(email string) bool {
    db.mu.RLock()
    defer db.mu.RUnlock()
    
    for _, user := range db.users {
        if strings.EqualFold(user.Email, email) {
            return true
        }
    }
    return false
}

func (db *MockDB) UsernameExists(username string) bool {
    db.mu.RLock()
    defer db.mu.RUnlock()
    
    return db.usernames[strings.ToLower(username)]
}

func (db *MockDB) AddUser(user User) {
    db.mu.Lock()
    defer db.mu.Unlock()
    
    db.users[user.ID] = user
    db.usernames[strings.ToLower(user.Username)] = true
}

// Main service that uses Queryfy with database validation
type UserService struct {
    db                *MockDB
    bannedDomains     map[string]bool
    reservedUsernames map[string]bool
    cache             *ValidationCache
    rateLimiter       *RateLimiter
}

type ValidationCache struct {
    emails    map[string]CacheEntry
    usernames map[string]CacheEntry
    mu        sync.RWMutex
    ttl       time.Duration
}

type CacheEntry struct {
    exists    bool
    timestamp time.Time
}

func NewValidationCache(ttl time.Duration) *ValidationCache {
    return &ValidationCache{
        emails:    make(map[string]CacheEntry),
        usernames: make(map[string]CacheEntry),
        ttl:       ttl,
    }
}

func (c *ValidationCache) GetEmail(email string) (exists bool, found bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    entry, ok := c.emails[email]
    if !ok || time.Since(entry.timestamp) > c.ttl {
        return false, false
    }
    return entry.exists, true
}

func (c *ValidationCache) SetEmail(email string, exists bool) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.emails[email] = CacheEntry{
        exists:    exists,
        timestamp: time.Now(),
    }
}

func main() {
    fmt.Println("=== Custom Validators with External Dependencies ===")
    fmt.Println("Demonstrating database validation, caching, and async patterns")
    fmt.Println()

    // Initialize service with dependencies
    db := NewMockDB()
    
    // Pre-populate some existing users
    db.AddUser(User{
        ID:       "user-001",
        Email:    "existing@example.com",
        Username: "existinguser",
    })
    db.AddUser(User{
        ID:       "user-002",
        Email:    "admin@company.com",
        Username: "admin",
    })
    
    userService := &UserService{
        db: db,
        bannedDomains: map[string]bool{
            "tempmail.com":      true,
            "guerrillamail.com": true,
            "throwaway.email":   true,
            "10minutemail.com":  true,
        },
        reservedUsernames: map[string]bool{
            "admin":     true,
            "root":      true,
            "system":    true,
            "moderator": true,
            "support":   true,
            "api":       true,
        },
        cache:       NewValidationCache(5 * time.Minute),
        rateLimiter: NewRateLimiter(10, time.Minute),
    }

    // EXAMPLE 1: Database Validation with Caching
    fmt.Println("Example 1: Database Uniqueness Validation with Caching")
    fmt.Println("------------------------------------------------------")
    demonstrateDatabaseValidation(userService)
    
    // EXAMPLE 2: Async-Style Validation
    fmt.Println("\nExample 2: Concurrent Validation Pattern")
    fmt.Println("----------------------------------------")
    demonstrateAsyncValidation(userService)
    
    // EXAMPLE 3: Rate-Limited Validation
    fmt.Println("\nExample 3: Rate-Limited API Validation")
    fmt.Println("--------------------------------------")
    demonstrateRateLimitedValidation(userService)
    
    // EXAMPLE 4: Bulk Validation
    fmt.Println("\nExample 4: Bulk User Import Validation")
    fmt.Println("--------------------------------------")
    demonstrateBulkValidation(userService)
    
    // EXAMPLE 5: Transaction-Style Validation
    fmt.Println("\nExample 5: Transaction-Style Validation")
    fmt.Println("---------------------------------------")
    demonstrateTransactionValidation(userService)
}

// Create registration schema with all database validations
func (s *UserService) CreateRegistrationSchema() *builders.ObjectSchema {
    return builders.Object().
        Field("email", builders.Transform(
            builders.String().Email().Required(),
        ).Add(transformers.Trim()).
          Add(transformers.Lowercase()).
          Add(builders.Transformer(s.validateEmailUniqueness)).
          Add(builders.Transformer(s.validateEmailDomain))).
        Field("username", builders.Transform(
            builders.String().
                MinLength(3).
                MaxLength(20).
                Pattern(`^[a-zA-Z0-9_]+$`).
                Required(),
        ).Add(transformers.Trim()).
          Add(transformers.Lowercase()).
          Add(builders.Transformer(s.validateUsernameAvailable))).
        Field("password", builders.String().
            MinLength(8).
            Custom(s.validatePasswordStrength).
            Required()).
        Field("age", builders.Number().
            Min(13).
            Max(120).
            Required()).
        Field("termsAccepted", builders.Bool().
            Custom(func(value interface{}) error {
                if accepted, ok := value.(bool); ok && !accepted {
                    return fmt.Errorf("terms must be accepted")
                }
                return nil
            }).
            Required())
}

// Email uniqueness with caching
func (s *UserService) validateEmailUniqueness(value interface{}) (interface{}, error) {
    email := value.(string)
    
    // Check cache first
    if exists, found := s.cache.GetEmail(email); found {
        if exists {
            return value, fmt.Errorf("email already registered")
        }
        return value, nil
    }
    
    // Check database
    exists := s.db.EmailExists(email)
    
    // Update cache
    s.cache.SetEmail(email, exists)
    
    if exists {
        return value, fmt.Errorf("email already registered")
    }
    
    return value, nil
}

// Domain validation
func (s *UserService) validateEmailDomain(value interface{}) (interface{}, error) {
    email := value.(string)
    parts := strings.Split(email, "@")
    if len(parts) != 2 {
        return value, nil // Let email validator handle this
    }
    
    domain := strings.ToLower(parts[1])
    if s.bannedDomains[domain] {
        return value, fmt.Errorf("disposable email addresses are not allowed")
    }
    
    return value, nil
}

// Username availability
func (s *UserService) validateUsernameAvailable(value interface{}) (interface{}, error) {
    username := value.(string)
    usernameLower := strings.ToLower(username)
    
    // Check reserved names
    if s.reservedUsernames[usernameLower] {
        return value, fmt.Errorf("username '%s' is reserved", username)
    }
    
    // Check database
    if s.db.UsernameExists(username) {
        return value, fmt.Errorf("username already taken")
    }
    
    return value, nil
}

// Password strength validation
func (s *UserService) validatePasswordStrength(value interface{}) error {
    password := value.(string)
    
    var hasUpper, hasLower, hasDigit, hasSpecial bool
    var commonPasswords = map[string]bool{
        "password":  true,
        "12345678":  true,
        "qwerty123": true,
        "admin123":  true,
    }
    
    // Check common passwords
    if commonPasswords[strings.ToLower(password)] {
        return fmt.Errorf("password is too common")
    }
    
    // Check character requirements
    for _, char := range password {
        switch {
        case 'A' <= char && char <= 'Z':
            hasUpper = true
        case 'a' <= char && char <= 'z':
            hasLower = true
        case '0' <= char && char <= '9':
            hasDigit = true
        case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
            hasSpecial = true
        }
    }
    
    if !hasUpper {
        return fmt.Errorf("password must contain at least one uppercase letter")
    }
    if !hasLower {
        return fmt.Errorf("password must contain at least one lowercase letter")
    }
    if !hasDigit {
        return fmt.Errorf("password must contain at least one digit")
    }
    if !hasSpecial {
        return fmt.Errorf("password must contain at least one special character")
    }
    
    return nil
}

// Demonstrate database validation
func demonstrateDatabaseValidation(userService *UserService) {
    registrationSchema := userService.CreateRegistrationSchema()
    
    testCases := []struct {
        name     string
        data     map[string]interface{}
        expected string
    }{
        {
            name: "Valid new user",
            data: map[string]interface{}{
                "email":         "newuser@example.com",
                "username":      "newuser123",
                "password":      "SecurePass123!",
                "age":           25,
                "termsAccepted": true,
            },
            expected: "success",
        },
        {
            name: "Existing email",
            data: map[string]interface{}{
                "email":         "existing@example.com",
                "username":      "different",
                "password":      "SecurePass123!",
                "age":           25,
                "termsAccepted": true,
            },
            expected: "email already registered",
        },
        {
            name: "Reserved username",
            data: map[string]interface{}{
                "email":         "another@example.com",
                "username":      "admin",
                "password":      "SecurePass123!",
                "age":           25,
                "termsAccepted": true,
            },
            expected: "username 'admin' is reserved",
        },
        {
            name: "Disposable email",
            data: map[string]interface{}{
                "email":         "test@tempmail.com",
                "username":      "validuser",
                "password":      "SecurePass123!",
                "age":           25,
                "termsAccepted": true,
            },
            expected: "disposable email addresses are not allowed",
        },
        {
            name: "Weak password",
            data: map[string]interface{}{
                "email":         "user@example.com",
                "username":      "user123",
                "password":      "password",
                "age":           25,
                "termsAccepted": true,
            },
            expected: "password must contain at least one uppercase letter",
        },
    }
    
    for _, tc := range testCases {
        fmt.Printf("\nTesting: %s\n", tc.name)
        ctx := qf.NewValidationContext(qf.Strict)
        
        // Use ValidateAndTransform to get cleaned data
        cleanData, err := registrationSchema.ValidateAndTransform(tc.data, ctx)
        
        if tc.expected == "success" {
            if err == nil {
                fmt.Println("✅ Validation passed")
                // Show transformed data
                if data, ok := cleanData.(map[string]interface{}); ok {
                    fmt.Printf("   Email (cleaned): %s\n", data["email"])
                    fmt.Printf("   Username (cleaned): %s\n", data["username"])
                }
            } else {
                fmt.Printf("❌ Unexpected error: %v\n", err)
            }
        } else {
            if err != nil {
                errorFound := false
                for _, e := range ctx.Errors() {
                    if strings.Contains(e.Message, tc.expected) {
                        fmt.Printf("✅ Expected error: %s\n", e.Message)
                        errorFound = true
                        break
                    }
                }
                if !errorFound {
                    fmt.Printf("❌ Wrong error. Expected: %s, Got: %v\n", tc.expected, err)
                }
            } else {
                fmt.Printf("❌ Expected error but validation passed\n")
            }
        }
    }
}

// Async validation implementation
type AsyncValidator struct {
    service *UserService
}

func NewAsyncValidator(service *UserService) *AsyncValidator {
    return &AsyncValidator{service: service}
}

func (av *AsyncValidator) ValidateUserAsync(ctx context.Context, data map[string]interface{}) error {
    // Pre-flight async checks
    results := make(chan error, 3)
    
    // Check email availability
    go func() {
        email, _ := data["email"].(string)
        if email == "" {
            results <- nil
            return
        }
        
        // Simulate network latency
        select {
        case <-time.After(50 * time.Millisecond):
            if av.service.db.EmailExists(email) {
                results <- fmt.Errorf("email already exists")
            } else {
                results <- nil
            }
        case <-ctx.Done():
            results <- ctx.Err()
        }
    }()
    
    // Check username availability
    go func() {
        username, _ := data["username"].(string)
        if username == "" {
            results <- nil
            return
        }
        
        // Simulate API call
        select {
        case <-time.After(75 * time.Millisecond):
            if av.service.db.UsernameExists(username) {
                results <- fmt.Errorf("username not available")
            } else {
                results <- nil
            }
        case <-ctx.Done():
            results <- ctx.Err()
        }
    }()
    
    // Check against external blacklist service
    go func() {
        email, _ := data["email"].(string)
        if email == "" {
            results <- nil
            return
        }
        
        domain := strings.Split(email, "@")[1]
        
        // Simulate external API call
        select {
        case <-time.After(100 * time.Millisecond):
            // Mock blacklist check
            blacklisted := map[string]bool{
                "spam.com":     true,
                "malicious.org": true,
            }
            if blacklisted[domain] {
                results <- fmt.Errorf("email domain is blacklisted")
            } else {
                results <- nil
            }
        case <-ctx.Done():
            results <- ctx.Err()
        }
    }()
    
    // Collect results
    for i := 0; i < 3; i++ {
        select {
        case err := <-results:
            if err != nil {
                return err
            }
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    
    // Now run synchronous Queryfy validation
    schema := av.service.CreateRegistrationSchema()
    return qf.Validate(data, schema)
}

func demonstrateAsyncValidation(userService *UserService) {
    validator := NewAsyncValidator(userService)
    
    testData := map[string]interface{}{
        "email":         "async@example.com",
        "username":      "asyncuser",
        "password":      "AsyncPass123!",
        "age":           30,
        "termsAccepted": true,
    }
    
    fmt.Println("Running async validation with timeout...")
    
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    start := time.Now()
    err := validator.ValidateUserAsync(ctx, testData)
    duration := time.Since(start)
    
    if err != nil {
        fmt.Printf("❌ Async validation failed: %v\n", err)
    } else {
        fmt.Printf("✅ Async validation passed in %v\n", duration)
    }
    
    // Test with existing user
    testData["email"] = "existing@example.com"
    
    fmt.Println("\nTesting with existing email...")
    start = time.Now()
    err = validator.ValidateUserAsync(context.Background(), testData)
    duration = time.Since(start)
    
    if err != nil {
        fmt.Printf("✅ Expected error: %v (in %v)\n", err, duration)
    } else {
        fmt.Printf("❌ Should have failed for existing email\n")
    }
}

// Rate limiter implementation
type RateLimiter struct {
    requests map[string][]time.Time
    limit    int
    window   time.Duration
    mu       sync.Mutex
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    rl := &RateLimiter{
        requests: make(map[string][]time.Time),
        limit:    limit,
        window:   window,
    }
    
    // Cleanup goroutine
    go func() {
        ticker := time.NewTicker(window)
        defer ticker.Stop()
        
        for range ticker.C {
            rl.cleanup()
        }
    }()
    
    return rl
}

func (rl *RateLimiter) Allow(key string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    now := time.Now()
    windowStart := now.Add(-rl.window)
    
    // Filter requests within window
    var validRequests []time.Time
    for _, t := range rl.requests[key] {
        if t.After(windowStart) {
            validRequests = append(validRequests, t)
        }
    }
    
    if len(validRequests) >= rl.limit {
        rl.requests[key] = validRequests
        return false
    }
    
    rl.requests[key] = append(validRequests, now)
    return true
}

func (rl *RateLimiter) cleanup() {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    now := time.Now()
    windowStart := now.Add(-rl.window)
    
    for key, requests := range rl.requests {
        var validRequests []time.Time
        for _, t := range requests {
            if t.After(windowStart) {
                validRequests = append(validRequests, t)
            }
        }
        
        if len(validRequests) == 0 {
            delete(rl.requests, key)
        } else {
            rl.requests[key] = validRequests
        }
    }
}

func demonstrateRateLimitedValidation(userService *UserService) {
    // Create API key validation schema
    apiSchema := builders.Object().
        Field("apiKey", builders.String().
            Pattern(`^sk_[a-zA-Z0-9]{32}$`).
            Custom(func(value interface{}) error {
                key := value.(string)
                
                // Check rate limit
                if !userService.rateLimiter.Allow(key) {
                    return fmt.Errorf("rate limit exceeded (max 10 requests per minute)")
                }
                
                // Validate API key (mock check)
                validKeys := map[string]bool{
                    "sk_12345678901234567890123456789012": true,
                    "sk_abcdefghijklmnopqrstuvwxyz123456": true,
                }
                
                if !validKeys[key] {
                    return fmt.Errorf("invalid API key")
                }
                
                return nil
            }).
            Required()).
        Field("endpoint", builders.String().
            Enum("/users", "/orders", "/products").
            Required()).
        Field("method", builders.String().
            Enum("GET", "POST", "PUT", "DELETE").
            Required())
    
    validKey := "sk_12345678901234567890123456789012"
    
    fmt.Println("Testing rate limiting (10 requests per minute)...")
    
    // Make requests
    for i := 1; i <= 12; i++ {
        request := map[string]interface{}{
            "apiKey":   validKey,
            "endpoint": "/users",
            "method":   "GET",
        }
        
        ctx := qf.NewValidationContext(qf.Strict)
        err := apiSchema.Validate(request, ctx)
        
        if err != nil {
            if i <= 10 {
                fmt.Printf("Request %d: ❌ Unexpected error: %v\n", i, ctx.Errors()[0].Message)
            } else {
                fmt.Printf("Request %d: ✅ Rate limited as expected\n", i)
            }
        } else {
            if i <= 10 {
                fmt.Printf("Request %d: ✅ Allowed\n", i)
            } else {
                fmt.Printf("Request %d: ❌ Should have been rate limited\n", i)
            }
        }
        
        // Small delay between requests
        time.Sleep(10 * time.Millisecond)
    }
}

// Bulk validation for imports
type BulkValidator struct {
    service       *UserService
    batchSize     int
    maxConcurrent int
}

func NewBulkValidator(service *UserService) *BulkValidator {
    return &BulkValidator{
        service:       service,
        batchSize:     100,
        maxConcurrent: 5,
    }
}

type ValidationResult struct {
    Index int
    Data  map[string]interface{}
    Error error
}

func (bv *BulkValidator) ValidateBatch(users []map[string]interface{}) []ValidationResult {
    results := make([]ValidationResult, len(users))
    resultChan := make(chan ValidationResult, len(users))
    
    // Worker pool
    sem := make(chan struct{}, bv.maxConcurrent)
    var wg sync.WaitGroup
    
    schema := bv.service.CreateRegistrationSchema()
    
    for i, user := range users {
        wg.Add(1)
        go func(index int, userData map[string]interface{}) {
            defer wg.Done()
            
            sem <- struct{}{} // Acquire semaphore
            defer func() { <-sem }() // Release semaphore
            
            ctx := qf.NewValidationContext(qf.Strict)
            cleanData, err := schema.ValidateAndTransform(userData, ctx)
            
            result := ValidationResult{
                Index: index,
                Data:  userData,
            }
            
            if err != nil {
                // Collect all errors
                var errors []string
                for _, e := range ctx.Errors() {
                    errors = append(errors, fmt.Sprintf("%s: %s", e.Path, e.Message))
                }
                result.Error = fmt.Errorf(strings.Join(errors, "; "))
            } else {
                result.Data = cleanData.(map[string]interface{})
            }
            
            resultChan <- result
        }(i, user)
    }
    
    // Wait for all workers
    go func() {
        wg.Wait()
        close(resultChan)
    }()
    
    // Collect results
    for result := range resultChan {
        results[result.Index] = result
    }
    
    return results
}

func demonstrateBulkValidation(userService *UserService) {
    bulkValidator := NewBulkValidator(userService)
    
    // Simulate bulk import data
    importData := []map[string]interface{}{
        {
            "email":         "user1@example.com",
            "username":      "user1",
            "password":      "Pass123!@#",
            "age":           25,
            "termsAccepted": true,
        },
        {
            "email":         "user2@tempmail.com", // Invalid domain
            "username":      "user2",
            "password":      "Pass123!@#",
            "age":           30,
            "termsAccepted": true,
        },
        {
            "email":         "user3@example.com",
            "username":      "admin", // Reserved
            "password":      "Pass123!@#",
            "age":           28,
            "termsAccepted": true,
        },
        {
            "email":         "existing@example.com", // Already exists
            "username":      "user4",
            "password":      "Pass123!@#",
            "age":           35,
            "termsAccepted": true,
        },
        {
            "email":         "user5@example.com",
            "username":      "user5",
            "password":      "weak", // Weak password
            "age":           22,
            "termsAccepted": true,
        },
    }
    
    fmt.Printf("Validating batch of %d users...\n", len(importData))
    
    start := time.Now()
    results := bulkValidator.ValidateBatch(importData)
    duration := time.Since(start)
    
    validCount := 0
    for _, result := range results {
        if result.Error == nil {
            validCount++
            fmt.Printf("✅ Row %d: Valid\n", result.Index+1)
        } else {
            fmt.Printf("❌ Row %d: %v\n", result.Index+1, result.Error)
        }
    }
    
    fmt.Printf("\nBulk validation complete in %v\n", duration)
    fmt.Printf("Valid: %d/%d\n", validCount, len(importData))
}

// Transaction-style validation
type ValidationTransaction struct {
    service  *UserService
    pending  []User
    rollback func()
}

func (s *UserService) BeginValidation() *ValidationTransaction {
    return &ValidationTransaction{
        service: s,
        pending: []User{},
    }
}

func (vt *ValidationTransaction) ValidateUser(data map[string]interface{}) error {
    schema := vt.service.CreateRegistrationSchema()
    
    ctx := qf.NewValidationContext(qf.Strict)
    cleanData, err := schema.ValidateAndTransform(data, ctx)
    if err != nil {
        return err
    }
    
    // Add to pending
    userData := cleanData.(map[string]interface{})
    user := User{
        ID:        fmt.Sprintf("user-%d", time.Now().UnixNano()),
        Email:     userData["email"].(string),
        Username:  userData["username"].(string),
        CreatedAt: time.Now(),
    }
    vt.pending = append(vt.pending, user)
    
    // Temporarily add to database for validation
    vt.service.db.AddUser(user)
    
    return nil
}

func (vt *ValidationTransaction) Commit() error {
    // In a real system, this would commit to the actual database
    fmt.Printf("Committing %d users to database\n", len(vt.pending))
    return nil
}

func (vt *ValidationTransaction) Rollback() {
    // Remove temporarily added users
    for _, user := range vt.pending {
        vt.service.db.mu.Lock()
        delete(vt.service.db.users, user.ID)
        delete(vt.service.db.usernames, strings.ToLower(user.Username))
        vt.service.db.mu.Unlock()
    }
    vt.pending = nil
}

func demonstrateTransactionValidation(userService *UserService) {
    fmt.Println("Starting validation transaction...")
    
    tx := userService.BeginValidation()
    defer tx.Rollback() // Ensure cleanup
    
    users := []map[string]interface{}{
        {
            "email":         "tx1@example.com",
            "username":      "txuser1",
            "password":      "TxPass123!",
            "age":           25,
            "termsAccepted": true,
        },
        {
            "email":         "tx2@example.com",
            "username":      "txuser2",
            "password":      "TxPass123!",
            "age":           30,
            "termsAccepted": true,
        },
        {
            "email":         "tx1@example.com", // Duplicate within transaction
            "username":      "txuser3",
            "password":      "TxPass123!",
            "age":           28,
            "termsAccepted": true,
        },
    }
    
    allValid := true
    for i, userData := range users {
        fmt.Printf("\nValidating user %d...\n", i+1)
        if err := tx.ValidateUser(userData); err != nil {
            fmt.Printf("❌ Validation failed: %v\n", err)
            allValid = false
            break
        } else {
            fmt.Printf("✅ User %d validated\n", i+1)
        }
    }
    
    if allValid {
        fmt.Println("\n✅ All validations passed, committing transaction")
        tx.Commit()
    } else {
        fmt.Println("\n❌ Validation failed, rolling back transaction")
        tx.Rollback()
    }
}