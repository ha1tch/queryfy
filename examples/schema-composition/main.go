package main

import (
    "fmt"
    "log"
    "strings"
    
    qf "github.com/ha1tch/queryfy"
    "github.com/ha1tch/queryfy/builders"
    "github.com/ha1tch/queryfy/builders/transformers"
)

func main() {
    fmt.Println("=== Schema Composition Patterns ===")
    fmt.Println("How to build reusable, composable schemas in Queryfy")
    fmt.Println()

    // PATTERN 1: Builder Functions
    fmt.Println("Pattern 1: Builder Functions for Reusable Fields")
    fmt.Println("------------------------------------------------")
    demonstrateBuilderFunctions()
    
    // PATTERN 2: Schema Factory
    fmt.Println("\nPattern 2: Schema Factory for Dynamic Composition")
    fmt.Println("-------------------------------------------------")
    demonstrateSchemaFactory()
    
    // PATTERN 3: Create vs Update Schemas
    fmt.Println("\nPattern 3: Different Schemas for Create vs Update")
    fmt.Println("-------------------------------------------------")
    demonstrateCreateVsUpdate()
    
    // PATTERN 4: Conditional Schema Building
    fmt.Println("\nPattern 4: Conditional Schema Building")
    fmt.Println("--------------------------------------")
    demonstrateConditionalSchemas()
    
    // PATTERN 5: Mixin Pattern
    fmt.Println("\nPattern 5: Mixin Pattern for Feature Sets")
    fmt.Println("-----------------------------------------")
    demonstrateMixinPattern()
    
    // PATTERN 6: Schema Inheritance
    fmt.Println("\nPattern 6: Schema Inheritance Pattern")
    fmt.Println("-------------------------------------")
    demonstrateInheritancePattern()
}

// PATTERN 1: Builder Functions
func demonstrateBuilderFunctions() {
    // Define reusable field groups as functions
    withTimestamps := func(obj *builders.ObjectSchema) *builders.ObjectSchema {
        return obj.
            Field("createdAt", builders.DateTime().Required()).
            Field("updatedAt", builders.DateTime().Required())
    }
    
    withUserFields := func(obj *builders.ObjectSchema) *builders.ObjectSchema {
        return obj.
            Field("email", builders.Transform(
                builders.String().Email().Required(),
            ).Add(transformers.Trim()).Add(transformers.Lowercase())).
            Field("username", builders.Transform(
                builders.String().MinLength(3).MaxLength(20).Required(),
            ).Add(transformers.Trim()).Add(transformers.Lowercase()))
    }
    
    withAddressFields := func(obj *builders.ObjectSchema) *builders.ObjectSchema {
        return obj.
            Field("street", builders.String().Required()).
            Field("city", builders.String().Required()).
            Field("state", builders.String().Length(2).Required()).
            Field("zipCode", builders.String().Pattern(`^\d{5}(-\d{4})?$`).Required())
    }
    
    withPhoneFields := func(obj *builders.ObjectSchema) *builders.ObjectSchema {
        return obj.
            Field("phoneNumber", builders.Transform(
                builders.String().Required(),
            ).Add(transformers.NormalizePhone("US"))).
            Field("phoneType", builders.String().Enum("mobile", "home", "work").Required())
    }
    
    // Compose different schemas using builder functions
    userSchema := withTimestamps(withUserFields(builders.Object()))
    
    customerSchema := withTimestamps(
        withUserFields(
            withAddressFields(
                withPhoneFields(
                    builders.Object().
                        Field("loyaltyPoints", builders.Number().Min(0).Required()).
                        Field("preferredContact", builders.String().Enum("email", "phone", "mail").Required()),
                ),
            ),
        ),
    )
    
    employeeSchema := withTimestamps(
        withUserFields(
            builders.Object().
                Field("employeeId", builders.String().Pattern(`^EMP\d{6}$`).Required()).
                Field("department", builders.String().Required()).
                Field("manager", builders.String().Optional()).
                Field("salary", builders.Number().Min(0).Required()),
        ),
    )
    
    // Test the schemas
    fmt.Println("Testing composed schemas:")
    
    // Test user data
    userData := map[string]interface{}{
        "email":     "john@example.com",
        "username":  "johndoe",
        "createdAt": "2024-01-15T10:00:00Z",
        "updatedAt": "2024-01-15T10:00:00Z",
    }
    
    ctx := qf.NewValidationContext(qf.Strict)
    if err := userSchema.Validate(userData, ctx); err == nil {
        fmt.Println("✅ Basic user schema validated successfully")
    }
    
    // Test customer data (should fail - missing required fields)
    if err := customerSchema.Validate(userData, ctx); err != nil {
        fmt.Println("❌ Customer schema failed (expected - missing address fields)")
    }
    
    // Complete customer data
    customerData := map[string]interface{}{
        "email":            "jane@example.com",
        "username":         "janedoe",
        "street":           "123 Main St",
        "city":             "Springfield",
        "state":            "IL",
        "zipCode":          "62701",
        "phoneNumber":      "555-123-4567",
        "phoneType":        "mobile",
        "loyaltyPoints":    1500,
        "preferredContact": "email",
        "createdAt":        "2024-01-15T10:00:00Z",
        "updatedAt":        "2024-01-15T10:00:00Z",
    }
    
    ctx = qf.NewValidationContext(qf.Strict)
    if err := customerSchema.Validate(customerData, ctx); err == nil {
        fmt.Println("✅ Customer schema validated successfully")
    }
}

// PATTERN 2: Schema Factory
type SchemaFactory struct {
    fields      map[string]qf.Schema
    required    map[string]bool
    transforms  map[string][]builders.Transformer
}

func NewSchemaFactory() *SchemaFactory {
    return &SchemaFactory{
        fields:     make(map[string]qf.Schema),
        required:   make(map[string]bool),
        transforms: make(map[string][]builders.Transformer),
    }
}

func (sf *SchemaFactory) AddField(name string, schema qf.Schema, required bool) *SchemaFactory {
    sf.fields[name] = schema
    sf.required[name] = required
    return sf
}

func (sf *SchemaFactory) AddFieldWithTransform(name string, schema qf.Schema, required bool, transforms ...builders.Transformer) *SchemaFactory {
    sf.fields[name] = schema
    sf.required[name] = required
    sf.transforms[name] = transforms
    return sf
}

func (sf *SchemaFactory) RemoveField(name string) *SchemaFactory {
    delete(sf.fields, name)
    delete(sf.required, name)
    delete(sf.transforms, name)
    return sf
}

func (sf *SchemaFactory) Clone() *SchemaFactory {
    newFactory := NewSchemaFactory()
    for k, v := range sf.fields {
        newFactory.fields[k] = v
        newFactory.required[k] = sf.required[k]
        if transforms, ok := sf.transforms[k]; ok {
            newFactory.transforms[k] = append([]builders.Transformer{}, transforms...)
        }
    }
    return newFactory
}

func (sf *SchemaFactory) Build() *builders.ObjectSchema {
    obj := builders.Object()
    for name, schema := range sf.fields {
        // Apply transforms if any
        if transforms, ok := sf.transforms[name]; ok && len(transforms) > 0 {
            transformSchema := builders.Transform(schema)
            for _, t := range transforms {
                transformSchema = transformSchema.Add(t)
            }
            schema = transformSchema
        }
        
        // Set required/optional
        if sf.required[name] {
            if rs, ok := schema.(interface{ Required() }); ok {
                rs.Required()
            }
        } else {
            if os, ok := schema.(interface{ Optional() }); ok {
                os.Optional()
            }
        }
        
        obj.Field(name, schema)
    }
    return obj
}

func demonstrateSchemaFactory() {
    // Base user fields
    baseFactory := NewSchemaFactory().
        AddFieldWithTransform("id", builders.String(), true).
        AddFieldWithTransform("email", builders.String().Email(), true,
            transformers.Trim(), transformers.Lowercase()).
        AddFieldWithTransform("name", builders.String().MinLength(2), true,
            transformers.Trim(), transformers.NormalizeWhitespace())
    
    // Admin extends base
    adminFactory := baseFactory.Clone().
        AddField("role", builders.String().Enum("admin", "superadmin"), true).
        AddField("permissions", builders.Array().Items(builders.String()), true).
        AddField("accessLevel", builders.Number().Min(1).Max(10), true)
    
    // Customer extends base differently
    customerFactory := baseFactory.Clone().
        RemoveField("id"). // Customers use email as ID
        AddField("tier", builders.String().Enum("bronze", "silver", "gold"), true).
        AddField("creditLimit", builders.Number().Min(0), true).
        AddField("referralCode", builders.String().Optional(), false)
    
    // Build schemas
    adminSchema := adminFactory.Build()
    customerSchema := customerFactory.Build()
    
    fmt.Println("Factory Pattern Results:")
    fmt.Println("- Base has: id, email, name")
    fmt.Println("- Admin adds: role, permissions, accessLevel")
    fmt.Println("- Customer removes id, adds: tier, creditLimit, referralCode")
    
    // Test admin data
    adminData := map[string]interface{}{
        "id":          "admin-001",
        "email":       "  ADMIN@COMPANY.COM  ",
        "name":        "  System   Administrator  ",
        "role":        "superadmin",
        "permissions": []interface{}{"read", "write", "delete"},
        "accessLevel": 10,
    }
    
    ctx := qf.NewValidationContext(qf.Strict)
    cleanAdmin, err := adminSchema.ValidateAndTransform(adminData, ctx)
    if err == nil {
        fmt.Println("✅ Admin schema validated and transformed")
        data := cleanAdmin.(map[string]interface{})
        fmt.Printf("   Cleaned email: %s\n", data["email"])
        fmt.Printf("   Cleaned name: %s\n", data["name"])
    }
}

// PATTERN 3: Create vs Update Schemas
func demonstrateCreateVsUpdate() {
    // Shared field builders
    emailField := func(required bool) qf.Schema {
        s := builders.Transform(
            builders.String().Email(),
        ).Add(transformers.Trim()).Add(transformers.Lowercase())
        
        if required {
            return s.Required()
        }
        return s.Optional()
    }
    
    passwordField := func(required bool, minLength int) qf.Schema {
        s := builders.String().
            MinLength(minLength).
            Pattern(`[A-Z]`).    // Has uppercase
            Pattern(`[a-z]`).    // Has lowercase
            Pattern(`[0-9]`)     // Has digit
            
        if required {
            return s.Required()
        }
        return s.Optional()
    }
    
    // Create schema - all fields required
    createUserSchema := builders.Object().
        Field("email", emailField(true)).
        Field("password", passwordField(true, 8)).
        Field("username", builders.Transform(
            builders.String().MinLength(3).Required(),
        ).Add(transformers.Trim()).Add(transformers.Lowercase())).
        Field("firstName", builders.String().MinLength(1).Required()).
        Field("lastName", builders.String().MinLength(1).Required()).
        Field("age", builders.Number().Min(13).Max(120).Required())
    
    // Update schema - all fields optional, but need current password
    updateUserSchema := builders.Object().
        Field("email", emailField(false)).
        Field("newPassword", passwordField(false, 8)).
        Field("username", builders.Transform(
            builders.String().MinLength(3).Optional(),
        ).Add(transformers.Trim()).Add(transformers.Lowercase())).
        Field("firstName", builders.String().MinLength(1).Optional()).
        Field("lastName", builders.String().MinLength(1).Optional()).
        Field("age", builders.Number().Min(13).Max(120).Optional()).
        Field("currentPassword", builders.String().Required()) // Always required for updates
    
    // Password change schema - specific operation
    changePasswordSchema := builders.Object().
        Field("currentPassword", builders.String().Required()).
        Field("newPassword", passwordField(true, 10)). // Stronger requirement
        Field("confirmPassword", builders.String().Required())
    
    fmt.Println("Schema Variations:")
    fmt.Println("- Create: All user fields required")
    fmt.Println("- Update: All fields optional except currentPassword")
    fmt.Println("- Change Password: Only password-related fields")
    
    // Test partial update
    updateData := map[string]interface{}{
        "email":           "newemail@example.com",
        "firstName":       "Jane",
        "currentPassword": "oldpass123",
        // Note: not updating other fields
    }
    
    ctx := qf.NewValidationContext(qf.Strict)
    if err := updateUserSchema.Validate(updateData, ctx); err == nil {
        fmt.Println("✅ Partial update validated successfully")
    }
}

// PATTERN 4: Conditional Schema Building
type AppConfig struct {
    Region            string
    RequirePhone      bool
    RequireAddress    bool
    MinPasswordLength int
    MaxUploadSize     int64
    AllowedCountries  []string
    Features          map[string]bool
}

func buildDynamicUserSchema(config AppConfig) *builders.ObjectSchema {
    schema := builders.Object()
    
    // Always required fields
    schema.Field("email", builders.Transform(
        builders.String().Email().Required(),
    ).Add(transformers.Trim()).Add(transformers.Lowercase())).
        Field("password", builders.String().
            MinLength(config.MinPasswordLength).
            Required())
    
    // Region-specific fields
    switch config.Region {
    case "US":
        schema.Field("ssn", builders.String().
            Pattern(`^\d{3}-\d{2}-\d{4}$`).
            Required())
    case "EU":
        schema.Field("gdprConsent", builders.Bool().Required()).
            Field("dataRetentionConsent", builders.Bool().Required())
    case "UK":
        schema.Field("niNumber", builders.String().
            Pattern(`^[A-Z]{2}\d{6}[A-Z]$`).
            Required())
    }
    
    // Conditional fields
    if config.RequirePhone {
        schema.Field("phone", builders.Transform(
            builders.String().Required(),
        ).Add(transformers.NormalizePhone(config.Region)))
    }
    
    if config.RequireAddress {
        schema.Field("country", builders.String().
            Enum(config.AllowedCountries...).
            Required()).
            Field("address", builders.Object().
                Field("line1", builders.String().Required()).
                Field("line2", builders.String().Optional()).
                Field("city", builders.String().Required()).
                Field("postalCode", builders.String().Required()))
    }
    
    // Feature-based fields
    if config.Features["2FA"] {
        schema.Field("twoFactorEnabled", builders.Bool().Required()).
            Field("twoFactorMethod", builders.String().
                Enum("sms", "email", "app").
                Optional())
    }
    
    if config.Features["ProfilePicture"] {
        schema.Field("profilePicture", builders.Object().
            Field("url", builders.String().Optional()).
            Field("size", builders.Number().Max(float64(config.MaxUploadSize)).Optional()))
    }
    
    return schema
}

func demonstrateConditionalSchemas() {
    // US Configuration
    usConfig := AppConfig{
        Region:            "US",
        RequirePhone:      true,
        RequireAddress:    false,
        MinPasswordLength: 8,
        MaxUploadSize:     5 * 1024 * 1024, // 5MB
        Features: map[string]bool{
            "2FA":            true,
            "ProfilePicture": true,
        },
    }
    
    // EU Configuration
    euConfig := AppConfig{
        Region:            "EU",
        RequirePhone:      false,
        RequireAddress:    true,
        MinPasswordLength: 10,
        MaxUploadSize:     2 * 1024 * 1024, // 2MB
        AllowedCountries:  []string{"DE", "FR", "IT", "ES", "NL"},
        Features: map[string]bool{
            "2FA":            true,
            "ProfilePicture": false, // GDPR restrictions
        },
    }
    
    usSchema := buildDynamicUserSchema(usConfig)
    euSchema := buildDynamicUserSchema(euConfig)
    
    fmt.Println("Regional Schema Differences:")
    fmt.Println("- US: Requires SSN, phone, has profile pictures")
    fmt.Println("- EU: Requires GDPR consent, address, no profile pictures")
    
    // Test US data
    usData := map[string]interface{}{
        "email":           "user@example.com",
        "password":        "SecurePass1",
        "ssn":             "123-45-6789",
        "phone":           "555-123-4567",
        "twoFactorEnabled": true,
        "twoFactorMethod":  "app",
    }
    
    ctx := qf.NewValidationContext(qf.Strict)
    if err := usSchema.Validate(usData, ctx); err == nil {
        fmt.Println("✅ US user data validated successfully")
    } else {
        fmt.Println("❌ US validation failed:", err)
    }
}

// PATTERN 5: Mixin Pattern
type SchemaMixin func(*builders.ObjectSchema) *builders.ObjectSchema

// Create reusable mixins
func TimestampsMixin() SchemaMixin {
    return func(schema *builders.ObjectSchema) *builders.ObjectSchema {
        return schema.
            Field("createdAt", builders.DateTime().Required()).
            Field("updatedAt", builders.DateTime().Required()).
            Field("deletedAt", builders.DateTime().Optional())
    }
}

func AuditMixin() SchemaMixin {
    return func(schema *builders.ObjectSchema) *builders.ObjectSchema {
        return schema.
            Field("createdBy", builders.String().Required()).
            Field("updatedBy", builders.String().Required()).
            Field("version", builders.Number().Min(0).Required())
    }
}

func SoftDeleteMixin() SchemaMixin {
    return func(schema *builders.ObjectSchema) *builders.ObjectSchema {
        return schema.
            Field("isDeleted", builders.Bool().Required()).
            Field("deletedAt", builders.DateTime().Optional()).
            Field("deletedBy", builders.String().Optional())
    }
}

func SEOMixin() SchemaMixin {
    return func(schema *builders.ObjectSchema) *builders.ObjectSchema {
        return schema.
            Field("slug", builders.Transform(
                builders.String().Pattern(`^[a-z0-9-]+$`).Required(),
            ).Add(transformers.Lowercase())).
            Field("metaTitle", builders.String().MaxLength(60).Optional()).
            Field("metaDescription", builders.String().MaxLength(160).Optional()).
            Field("keywords", builders.Array().Items(builders.String()).Optional())
    }
}

// Apply multiple mixins
func applyMixins(schema *builders.ObjectSchema, mixins ...SchemaMixin) *builders.ObjectSchema {
    for _, mixin := range mixins {
        schema = mixin(schema)
    }
    return schema
}

func demonstrateMixinPattern() {
    // Blog post schema with mixins
    blogPostSchema := applyMixins(
        builders.Object().
            Field("title", builders.String().MinLength(1).MaxLength(200).Required()).
            Field("content", builders.String().MinLength(10).Required()).
            Field("status", builders.String().Enum("draft", "published", "archived").Required()),
        TimestampsMixin(),
        AuditMixin(),
        SEOMixin(),
    )
    
    // Product schema with different mixins
    productSchema := applyMixins(
        builders.Object().
            Field("name", builders.String().Required()).
            Field("price", builders.Number().Min(0).Required()).
            Field("inventory", builders.Number().Min(0).Required()),
        TimestampsMixin(),
        AuditMixin(),
        SoftDeleteMixin(),
    )
    
    fmt.Println("Mixin Pattern Results:")
    fmt.Println("- Blog Post has: content fields + timestamps + audit + SEO")
    fmt.Println("- Product has: product fields + timestamps + audit + soft delete")
    
    // Test blog post
    blogData := map[string]interface{}{
        "title":       "My Amazing Blog Post",
        "content":     "This is the content of my blog post...",
        "status":      "published",
        "slug":        "my-amazing-blog-post",
        "createdAt":   "2024-01-15T10:00:00Z",
        "updatedAt":   "2024-01-15T10:00:00Z",
        "createdBy":   "author123",
        "updatedBy":   "author123",
        "version":     1,
    }
    
    ctx := qf.NewValidationContext(qf.Strict)
    if err := blogPostSchema.Validate(blogData, ctx); err == nil {
        fmt.Println("✅ Blog post with mixins validated successfully")
    }
}

// PATTERN 6: Schema Inheritance
type BaseEntity struct {
    schema *builders.ObjectSchema
}

func NewBaseEntity() *BaseEntity {
    return &BaseEntity{
        schema: builders.Object().
            Field("id", builders.String().Pattern(`^[a-zA-Z0-9-]+$`).Required()).
            Field("createdAt", builders.DateTime().Required()).
            Field("updatedAt", builders.DateTime().Required()),
    }
}

func (b *BaseEntity) Extend() *ExtendedEntity {
    // Clone the base schema fields
    extended := builders.Object()
    // In real implementation, would copy fields from base
    // For demo, we'll manually add them
    extended.
        Field("id", builders.String().Pattern(`^[a-zA-Z0-9-]+$`).Required()).
        Field("createdAt", builders.DateTime().Required()).
        Field("updatedAt", builders.DateTime().Required())
    
    return &ExtendedEntity{
        BaseEntity: b,
        schema:     extended,
    }
}

type ExtendedEntity struct {
    *BaseEntity
    schema *builders.ObjectSchema
}

func (e *ExtendedEntity) AddField(name string, fieldSchema qf.Schema) *ExtendedEntity {
    e.schema.Field(name, fieldSchema)
    return e
}

func (e *ExtendedEntity) Build() *builders.ObjectSchema {
    return e.schema
}

func demonstrateInheritancePattern() {
    // Create base entity
    base := NewBaseEntity()
    
    // Create user entity extending base
    userEntity := base.Extend().
        AddField("email", builders.String().Email().Required()).
        AddField("username", builders.String().Required()).
        Build()
    
    // Create product entity extending base
    productEntity := base.Extend().
        AddField("name", builders.String().Required()).
        AddField("price", builders.Number().Min(0).Required()).
        AddField("sku", builders.String().Pattern(`^SKU-\d+$`).Required()).
        Build()
    
    fmt.Println("Inheritance Pattern Results:")
    fmt.Println("- Base Entity has: id, createdAt, updatedAt")
    fmt.Println("- User Entity inherits base + adds: email, username")
    fmt.Println("- Product Entity inherits base + adds: name, price, sku")
    
    // Test inherited schemas
    userData := map[string]interface{}{
        "id":        "user-123",
        "email":     "user@example.com",
        "username":  "johndoe",
        "createdAt": "2024-01-15T10:00:00Z",
        "updatedAt": "2024-01-15T10:00:00Z",
    }
    
    ctx := qf.NewValidationContext(qf.Strict)
    if err := userEntity.Validate(userData, ctx); err == nil {
        fmt.Println("✅ User entity with inheritance validated successfully")
    }
}

// Helper function to demonstrate real-world usage
func demonstrateRealWorldComposition() {
    fmt.Println("\n=== Real-World Example: E-commerce Platform ===")
    fmt.Println("----------------------------------------------")
    
    // Define common mixins for the platform
    entityMixin := func(schema *builders.ObjectSchema) *builders.ObjectSchema {
        return schema.
            Field("id", builders.String().Pattern(`^[A-Z]+\d{10}$`).Required()).
            Field("createdAt", builders.DateTime().Required()).
            Field("updatedAt", builders.DateTime().Required())
    }
    
    pricingMixin := func(schema *builders.ObjectSchema) *builders.ObjectSchema {
        return schema.
            Field("price", builders.Number().Min(0).Required()).
            Field("currency", builders.String().Enum("USD", "EUR", "GBP").Required()).
            Field("taxRate", builders.Number().Min(0).Max(1).Required())
    }
    
    // Product schema combining multiple patterns
    productSchema := applyMixins(
        builders.Object().
            Field("name", builders.String().MinLength(1).MaxLength(200).Required()).
            Field("description", builders.String().Required()).
            Field("category", builders.String().Required()).
            Field("inventory", builders.Number().Min(0).Required()),
        entityMixin,
        pricingMixin,
        SEOMixin(),
    )
    
    // Order schema with conditional fields
    createOrderSchema := func(isPremium bool) *builders.ObjectSchema {
        schema := applyMixins(
            builders.Object().
                Field("items", builders.Array().MinItems(1).Items(
                    builders.Object().
                        Field("productId", builders.String().Required()).
                        Field("quantity", builders.Number().Min(1).Required()),
                )).Required()).
                Field("shippingAddress", builders.Object().
                    Field("street", builders.String().Required()).
                    Field("city", builders.String().Required()).
                    Field("postalCode", builders.String().Required()).
                    Required()),
            entityMixin,
            pricingMixin,
        )
        
        if isPremium {
            schema.
                Field("expeditedShipping", builders.Bool().Optional()).
                Field("giftWrapping", builders.Bool().Optional()).
                Field("personalMessage", builders.String().MaxLength(500).Optional())
        }
        
        return schema
    }
    
    regularOrderSchema := createOrderSchema(false)
    premiumOrderSchema := createOrderSchema(true)
    
    fmt.Println("E-commerce Schemas Created:")
    fmt.Println("- Product: entity + pricing + SEO mixins")
    fmt.Println("- Regular Order: entity + pricing + basic fields")
    fmt.Println("- Premium Order: regular + premium features")
}

// Main execution includes the real-world example
func init() {
    // This would run after all the pattern demonstrations
    defer demonstrateRealWorldComposition()
}