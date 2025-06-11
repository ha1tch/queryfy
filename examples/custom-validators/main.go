package main

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
	"unicode"

	qf "github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
)

// Example: E-commerce platform with complex validation requirements
func main() {
	fmt.Println("=== Queryfy Custom Validators Example ===\n")

	// Run different examples
	userRegistrationExample()
	productListingExample()
	orderValidationExample()
	configurationExample()
}

// Example 1: User Registration with Custom Validators
func userRegistrationExample() {
	fmt.Println("1. User Registration Validation")
	fmt.Println("-------------------------------")

	// Custom validator: Username must not contain offensive words
	usernameValidator := builders.Custom(func(value interface{}) error {
		username, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string")
		}

		offensiveWords := []string{"admin", "root", "system", "null"}
		usernameLower := strings.ToLower(username)
		
		for _, word := range offensiveWords {
			if strings.Contains(usernameLower, word) {
				return fmt.Errorf("username contains restricted word: %s", word)
			}
		}

		// Check for special character balance
		if !isBalancedUsername(username) {
			return fmt.Errorf("username has unbalanced brackets or quotes")
		}

		return nil
	})

	// Custom validator: Password strength checker
	passwordStrengthValidator := func(value interface{}) error {
		password, _ := value.(string)
		
		strength := calculatePasswordStrength(password)
		if strength < 3 {
			return fmt.Errorf("password is too weak (strength: %d/5)", strength)
		}
		
		// Check against common patterns
		commonPatterns := []string{
			"^12345",             // sequential numbers
			"^qwerty",            // keyboard patterns
			"^password",          // obvious passwords
		}
		
		for _, pattern := range commonPatterns {
			if matched, _ := regexp.MatchString(pattern, strings.ToLower(password)); matched {
				return fmt.Errorf("password matches common pattern")
			}
		}
		
		return nil
	}

	// Custom validator: Email domain whitelist/blacklist
	emailDomainValidator := func(value interface{}) error {
		email, _ := value.(string)
		
		// Extract domain
		parts := strings.Split(email, "@")
		if len(parts) != 2 {
			return nil // Let the email validator handle this
		}
		
		domain := strings.ToLower(parts[1])
		
		// Blacklisted domains
		blacklist := []string{"tempmail.com", "throwaway.email", "guerrillamail.com"}
		for _, blocked := range blacklist {
			if domain == blocked {
				return fmt.Errorf("email domain %s is not allowed", domain)
			}
		}
		
		// For B2B, you might want to whitelist corporate domains
		// whitelist := []string{"company.com", "enterprise.org"}
		
		return nil
	}

	// Custom validator: Birth date validation
	birthDateValidator := func(value interface{}) error {
		dateStr, _ := value.(string)
		
		birthDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return fmt.Errorf("invalid date format, use YYYY-MM-DD")
		}
		
		age := calculateAge(birthDate)
		
		if age < 13 {
			return fmt.Errorf("must be at least 13 years old")
		}
		
		if age > 120 {
			return fmt.Errorf("invalid birth date")
		}
		
		// Check for suspicious dates
		if birthDate.Day() == 1 && birthDate.Month() == 1 {
			// Many people put Jan 1 as fake date
			return fmt.Errorf("please provide your actual birth date")
		}
		
		return nil
	}

	// Build the complete user schema
	userSchema := builders.Object().
		Field("username", builders.And(
			builders.String().MinLength(3).MaxLength(20).Pattern(`^[a-zA-Z0-9_\-\[\]{}]+$`),
			usernameValidator,
		).Required()).
		Field("email", builders.String().
			Email().
			Custom(emailDomainValidator).
			Required()).
		Field("password", builders.String().
			MinLength(8).
			MaxLength(100).
			Custom(passwordStrengthValidator).
			Required()).
		Field("confirmPassword", builders.String().Required()).
		Field("birthDate", builders.String().
			Custom(birthDateValidator).
			Required()).
		Field("phoneNumber", builders.String().
			Custom(phoneNumberValidator).
			Optional()).
		Field("referralCode", builders.String().
			Custom(referralCodeValidator).
			Optional())

	// Test data
	testUsers := []map[string]interface{}{
		{
			"username":        "john_doe",
			"email":          "john@company.com",
			"password":       "Str0ng!Pass#2024",
			"confirmPassword": "Str0ng!Pass#2024",
			"birthDate":      "1990-06-15",
			"phoneNumber":    "+1-555-123-4567",
			"referralCode":   "FRIEND2024",
		},
		{
			"username":        "admin_user", // Fails: contains "admin"
			"email":          "test@tempmail.com", // Fails: blacklisted domain
			"password":       "password123", // Fails: common pattern
			"confirmPassword": "password123",
			"birthDate":      "2015-01-01", // Fails: too young
		},
		{
			"username":        "user[with{unbalanced", // Fails: unbalanced brackets
			"email":          "valid@email.com",
			"password":       "aaaaaaaaa", // Fails: repeated character
			"confirmPassword": "aaaaaaaaa",
			"birthDate":      "2000-01-01", // Suspicious date
		},
	}

	for i, userData := range testUsers {
		fmt.Printf("\nValidating user %d:\n", i+1)
		
		// Additional validation: password confirmation
		if err := qf.Validate(userData, userSchema); err == nil {
			// Check password match
			if userData["password"] != userData["confirmPassword"] {
				fmt.Println("[X] Passwords do not match")
			} else {
				fmt.Println("[✓] All validations passed!")
			}
		} else {
			fmt.Printf("[X] Validation failed:\n%v\n", err)
		}
	}
}

// Example 2: Product Listing with Business Rules
func productListingExample() {
	fmt.Println("\n\n2. Product Listing Validation")
	fmt.Println("------------------------------")

	// Custom validator: SKU format validation
	skuValidator := func(value interface{}) error {
		sku, _ := value.(string)
		
		// SKU Format: CAT-BRAND-12345-SIZE
		parts := strings.Split(sku, "-")
		if len(parts) != 4 {
			return fmt.Errorf("SKU must have format: CATEGORY-BRAND-NUMBER-SIZE")
		}
		
		// Validate category
		validCategories := []string{"ELEC", "CLTH", "HOME", "FOOD", "BOOK"}
		if !contains(validCategories, parts[0]) {
			return fmt.Errorf("invalid category code")
		}
		
		// Validate brand (2-4 uppercase letters)
		brandPattern := `^[A-Z]{2,4}$`
		if matched, _ := regexp.MatchString(brandPattern, parts[1]); !matched {
			return fmt.Errorf("brand code must be 2-4 uppercase letters")
		}
		
		// Validate product number
		numberPattern := `^\d{5}$`
		if matched, _ := regexp.MatchString(numberPattern, parts[2]); !matched {
			return fmt.Errorf("product number must be 5 digits")
		}
		
		return nil
	}

	// Custom validator: Price consistency
	priceConsistencyValidator := func(value interface{}) error {
		product, ok := value.(map[string]interface{})
		if !ok {
			return nil
		}
		
		regularPrice, _ := product["regularPrice"].(float64)
		salePrice, hasSale := product["salePrice"].(float64)
		costPrice, _ := product["costPrice"].(float64)
		
		// Business rule: Minimum 20% margin
		minPrice := costPrice * 1.20
		if regularPrice < minPrice {
			return fmt.Errorf("regular price must be at least 20%% above cost")
		}
		
		// Business rule: Sale price must be less than regular price
		if hasSale && salePrice >= regularPrice {
			return fmt.Errorf("sale price must be less than regular price")
		}
		
		// Business rule: Maximum 70% discount
		if hasSale && salePrice < (regularPrice * 0.30) {
			return fmt.Errorf("discount cannot exceed 70%%")
		}
		
		return nil
	}

	// Custom validator: Image validation
	imageValidator := func(value interface{}) error {
		images, ok := value.([]interface{})
		if !ok || len(images) == 0 {
			return fmt.Errorf("at least one image is required")
		}
		
		for i, img := range images {
			imgMap, ok := img.(map[string]interface{})
			if !ok {
				continue
			}
			
			url, _ := imgMap["url"].(string)
			altText, _ := imgMap["altText"].(string)
			
			// Check image URL format
			if !strings.HasPrefix(url, "https://") {
				return fmt.Errorf("image %d: URL must use HTTPS", i+1)
			}
			
			// Check file extension
			validExts := []string{".jpg", ".jpeg", ".png", ".webp"}
			hasValidExt := false
			for _, ext := range validExts {
				if strings.HasSuffix(strings.ToLower(url), ext) {
					hasValidExt = true
					break
				}
			}
			if !hasValidExt {
				return fmt.Errorf("image %d: invalid file type", i+1)
			}
			
			// SEO: Alt text requirements
			if len(altText) < 10 {
				return fmt.Errorf("image %d: alt text too short for SEO", i+1)
			}
		}
		
		return nil
	}

	// Build product schema
	productSchema := builders.Object().
		Field("sku", builders.String().
			Custom(skuValidator).
			Required()).
		Field("title", builders.String().
			MinLength(10).
			MaxLength(200).
			Custom(func(value interface{}) error {
				title, _ := value.(string)
				// Title must contain at least 2 words
				words := strings.Fields(title)
				if len(words) < 2 {
					return fmt.Errorf("title must contain at least 2 words")
				}
				// No excessive caps
				upperCount := 0
				for _, r := range title {
					if unicode.IsUpper(r) {
						upperCount++
					}
				}
				if float64(upperCount)/float64(len(title)) > 0.5 {
					return fmt.Errorf("title has too many capital letters")
				}
				return nil
			}).
			Required()).
		Field("description", builders.String().
			MinLength(50).
			Custom(func(value interface{}) error {
				desc, _ := value.(string)
				// Check for keyword stuffing
				words := strings.Fields(strings.ToLower(desc))
				wordCount := make(map[string]int)
				for _, word := range words {
					wordCount[word]++
				}
				for word, count := range wordCount {
					if count > 5 && float64(count)/float64(len(words)) > 0.1 {
						return fmt.Errorf("keyword stuffing detected: '%s' appears too often", word)
					}
				}
				return nil
			}).
			Required()).
		Field("regularPrice", builders.Number().Min(0.01).Required()).
		Field("salePrice", builders.Number().Min(0.01).Optional()).
		Field("costPrice", builders.Number().Min(0).Required()).
		Field("inventory", builders.Object().
			Field("quantity", builders.Number().Min(0).Integer().Required()).
			Field("warehouse", builders.String().Enum("US-EAST", "US-WEST", "EU-CENTRAL").Required()).
			Field("reorderPoint", builders.Number().Min(0).Integer()).
			Custom(func(value interface{}) error {
				inv, _ := value.(map[string]interface{})
				qty, _ := inv["quantity"].(float64)
				reorder, _ := inv["reorderPoint"].(float64)
				
				if reorder > 0 && qty <= reorder {
					return fmt.Errorf("warning: quantity at or below reorder point")
				}
				return nil
			})).
		Field("images", builders.Array().
			Custom(imageValidator).
			Required()).
		Custom(priceConsistencyValidator) // Product-level validation

	// Test products
	products := []map[string]interface{}{
		{
			"sku":          "ELEC-SONY-12345-M",
			"title":        "Sony Wireless Headphones WH-1000XM5",
			"description":  "Experience exceptional sound quality with industry-leading noise cancellation. These premium wireless headphones deliver immersive audio for music lovers.",
			"regularPrice": 349.99,
			"salePrice":    299.99,
			"costPrice":    200.00,
			"inventory": map[string]interface{}{
				"quantity":     50,
				"warehouse":    "US-WEST",
				"reorderPoint": 20,
			},
			"images": []interface{}{
				map[string]interface{}{
					"url":     "https://cdn.example.com/products/sony-wh1000xm5-black.jpg",
					"altText": "Sony WH-1000XM5 Wireless Headphones in Black",
				},
			},
		},
		{
			"sku":          "INVALID-SKU", // Invalid format
			"title":        "AMAZING PRODUCT!!!", // Too many caps
			"description":  "Buy buy buy! Best best best! Amazing amazing amazing!", // Keyword stuffing
			"regularPrice": 100.00,
			"salePrice":    15.00, // Exceeds 70% discount
			"costPrice":    85.00, // Price doesn't meet margin requirement
			"inventory": map[string]interface{}{
				"quantity":  5,
				"warehouse": "US-EAST",
			},
			"images": []interface{}{
				map[string]interface{}{
					"url":     "http://insecure.com/image.gif", // Not HTTPS, wrong format
					"altText": "Product", // Too short
				},
			},
		},
	}

	for i, product := range products {
		fmt.Printf("\nValidating product %d:\n", i+1)
		if err := qf.Validate(product, productSchema); err != nil {
			fmt.Printf("[X] Validation failed:\n%v\n", err)
		} else {
			fmt.Println("[✓] Product validation passed!")
		}
	}
}

// Example 3: Order Processing with Complex Rules
func orderValidationExample() {
	fmt.Println("\n\n3. Order Processing Validation")
	fmt.Println("-------------------------------")

	// Custom validator: Shipping address validation
	addressValidator := func(value interface{}) error {
		addr, ok := value.(map[string]interface{})
		if !ok {
			return nil
		}
		
		country, _ := addr["country"].(string)
		state, _ := addr["state"].(string)
		zip, _ := addr["zip"].(string)
		
		// Country-specific validation
		switch country {
		case "US":
			// Validate US state code
			usStates := []string{"AL", "AK", "AZ", "AR", "CA", "CO", "CT", "DE", "FL", "GA",
				"HI", "ID", "IL", "IN", "IA", "KS", "KY", "LA", "ME", "MD",
				"MA", "MI", "MN", "MS", "MO", "MT", "NE", "NV", "NH", "NJ",
				"NM", "NY", "NC", "ND", "OH", "OK", "OR", "PA", "RI", "SC",
				"SD", "TN", "TX", "UT", "VT", "VA", "WA", "WV", "WI", "WY"}
			if !contains(usStates, state) {
				return fmt.Errorf("invalid US state code")
			}
			// Validate ZIP code
			usZipPattern := `^\d{5}(-\d{4})?$`
			if matched, _ := regexp.MatchString(usZipPattern, zip); !matched {
				return fmt.Errorf("invalid US ZIP code format")
			}
		case "CA":
			// Validate Canadian postal code
			caPostalPattern := `^[A-Z]\d[A-Z] \d[A-Z]\d$`
			if matched, _ := regexp.MatchString(caPostalPattern, zip); !matched {
				return fmt.Errorf("invalid Canadian postal code format")
			}
		case "UK":
			// Validate UK postcode
			ukPostcodePattern := `^[A-Z]{1,2}\d[A-Z\d]? ?\d[A-Z]{2}$`
			if matched, _ := regexp.MatchString(ukPostcodePattern, zip); !matched {
				return fmt.Errorf("invalid UK postcode format")
			}
		}
		
		return nil
	}

	// Custom validator: Payment method validation
	paymentValidator := func(value interface{}) error {
		payment, ok := value.(map[string]interface{})
		if !ok {
			return nil
		}
		
		method, _ := payment["method"].(string)
		
		switch method {
		case "credit_card":
			number, _ := payment["cardNumber"].(string)
			// Remove spaces and dashes
			number = strings.ReplaceAll(number, " ", "")
			number = strings.ReplaceAll(number, "-", "")
			
			// Luhn algorithm validation
			if !isValidCreditCard(number) {
				return fmt.Errorf("invalid credit card number")
			}
			
			// Check expiry
			expiry, _ := payment["expiry"].(string)
			expiryPattern := `^(0[1-9]|1[0-2])\/\d{2}$`
			if matched, _ := regexp.MatchString(expiryPattern, expiry); !matched {
				return fmt.Errorf("expiry must be in MM/YY format")
			}
			
			// Validate CVV
			cvv, _ := payment["cvv"].(string)
			cvvPattern := `^\d{3,4}$`
			if matched, _ := regexp.MatchString(cvvPattern, cvv); !matched {
				return fmt.Errorf("CVV must be 3 or 4 digits")
			}
			
		case "paypal":
			email, _ := payment["paypalEmail"].(string)
			if !strings.Contains(email, "@") {
				return fmt.Errorf("valid PayPal email required")
			}
			
		case "crypto":
			wallet, _ := payment["walletAddress"].(string)
			coin, _ := payment["cryptocurrency"].(string)
			
			switch coin {
			case "BTC":
				// Basic Bitcoin address validation
				btcPattern := `^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$`
				if matched, _ := regexp.MatchString(btcPattern, wallet); !matched {
					return fmt.Errorf("invalid Bitcoin address")
				}
			case "ETH":
				// Basic Ethereum address validation
				ethPattern := `^0x[a-fA-F0-9]{40}$`
				if matched, _ := regexp.MatchString(ethPattern, wallet); !matched {
					return fmt.Errorf("invalid Ethereum address")
				}
			}
		}
		
		return nil
	}

	// Custom validator: Order items validation
	orderItemsValidator := func(value interface{}) error {
		items, ok := value.([]interface{})
		if !ok || len(items) == 0 {
			return fmt.Errorf("order must contain at least one item")
		}
		
		totalAmount := 0.0
		skuMap := make(map[string]int)
		
		for i, item := range items {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			
			sku, _ := itemMap["sku"].(string)
			
			// Handle both int and float64 for quantity
			var quantity float64
			switch q := itemMap["quantity"].(type) {
			case float64:
				quantity = q
			case int:
				quantity = float64(q)
			default:
				quantity = 0
			}
			
			// Handle price
			price, _ := itemMap["price"].(float64)
			
			
			// Check for duplicate SKUs
			if _, exists := skuMap[sku]; exists {
				return fmt.Errorf("duplicate SKU found: %s", sku)
			}
			skuMap[sku] = i
			
			// Validate quantity limits
			if quantity > 100 {
				return fmt.Errorf("item %d: quantity exceeds maximum of 100", i+1)
			}
			
			// Calculate total
			totalAmount += price * quantity
		}
		
		// Business rule: Minimum order amount
		if totalAmount < 10.00 {
			return fmt.Errorf("minimum order amount is $10.00 (current: $%.2f)", totalAmount)
		}
		
		// Business rule: Maximum order amount for new customers
		if totalAmount > 5000.00 {
			return fmt.Errorf("orders over $5000 require manual approval")
		}
		
		return nil
	}

	// Build order schema
	orderSchema := builders.Object().
		Field("orderNumber", builders.String().
			Pattern(`^ORD-\d{10}$`).
			Required()).
		Field("customer", builders.Object().
			Field("id", builders.String().UUID().Required()).
			Field("email", builders.String().Email().Required()).
			Field("isNewCustomer", builders.Bool())).
		Field("items", builders.Array().
			Of(builders.Object().
				Field("sku", builders.String().Required()).
				Field("quantity", builders.Number().Min(1).Integer().Required()).
				Field("price", builders.Number().Min(0).Required())).
			MinItems(1).
			Required()).
		Field("shipping", builders.Object().
			Field("method", builders.String().
				Enum("standard", "express", "overnight").
				Required()).
			Field("address", builders.Object().
				Field("street", builders.String().Required()).
				Field("city", builders.String().Required()).
				Field("state", builders.String().Required()).
				Field("country", builders.String().
					Enum("US", "CA", "UK", "AU").
					Required()).
				Field("zip", builders.String().Required()).
				Custom(addressValidator).
				Required()).
			Field("instructions", builders.String().MaxLength(500))).
		Field("payment", builders.Object().
			Field("method", builders.String().
				Enum("credit_card", "paypal", "crypto", "purchase_order").
				Required()).
			Field("cardNumber", builders.String().Optional()).
			Field("expiry", builders.String().Optional()).
			Field("cvv", builders.String().Optional()).
			Field("paypalEmail", builders.String().Optional()).
			Field("walletAddress", builders.String().Optional()).
			Field("cryptocurrency", builders.String().Optional()).
			Custom(paymentValidator).
			Required()).
		Field("metadata", builders.Object().
			Field("source", builders.String().Enum("web", "mobile", "api")).
			Field("ipAddress", builders.String().
				Custom(ipAddressValidator)).
			Field("userAgent", builders.String())).
		Custom(func(value interface{}) error {
			// Order-level validation
			order, _ := value.(map[string]interface{})
			
			// First run the items validator
			items, _ := order["items"].([]interface{})
			if err := orderItemsValidator(items); err != nil {
				return err
			}
			
			// Check shipping method vs order total
			shipping, _ := order["shipping"].(map[string]interface{})
			shippingMethod, _ := shipping["method"].(string)
			
			var orderTotal float64
			for _, item := range items {
				itemMap, _ := item.(map[string]interface{})
				
				// Handle both int and float64 for quantity
				var qty float64
				switch q := itemMap["quantity"].(type) {
				case float64:
					qty = q
				case int:
					qty = float64(q)
				default:
					qty = 0
				}
				
				price, _ := itemMap["price"].(float64)
				orderTotal += qty * price
			}
			
			// Business rule: Free overnight shipping for orders over $500
			if orderTotal < 500 && shippingMethod == "overnight" {
				return fmt.Errorf("overnight shipping requires minimum order of $500")
			}
			
			return nil
		})

	// Test order
	order := map[string]interface{}{
		"orderNumber": "ORD-2024012345",
		"customer": map[string]interface{}{
			"id":            "550e8400-e29b-41d4-a716-446655440000",
			"email":         "customer@example.com",
			"isNewCustomer": false,
		},
		"items": []interface{}{
			map[string]interface{}{
				"sku":      "ELEC-SONY-12345-M",
				"quantity": 2,
				"price":    299.99,
			},
			map[string]interface{}{
				"sku":      "ELEC-BOSE-54321-L",
				"quantity": 1,
				"price":    199.99,
			},
		},
		"shipping": map[string]interface{}{
			"method": "express",
			"address": map[string]interface{}{
				"street":  "123 Main St, Apt 4B",
				"city":    "New York",
				"state":   "NY",
				"country": "US",
				"zip":     "10001",
			},
			"instructions": "Leave at front desk",
		},
		"payment": map[string]interface{}{
			"method":     "credit_card",
			"cardNumber": "4111-1111-1111-1111", // Test card number
			"expiry":     "12/25",
			"cvv":        "123",
		},
		"metadata": map[string]interface{}{
			"source":    "web",
			"ipAddress": "192.168.1.1",
			"userAgent": "Mozilla/5.0...",
		},
	}

	fmt.Printf("\nValidating order:\n")
	if err := qf.ValidateWithMode(order, orderSchema, qf.Loose); err != nil {
		fmt.Printf("[X] Order validation failed:\n%v\n", err)
	} else {
		fmt.Println("[✓] Order validation passed!")
		
		// Query some data
		total, _ := qf.Query(order, "items[0].price")
		method, _ := qf.Query(order, "payment.method")
		fmt.Printf("\nFirst item price: $%.2f\n", total)
		fmt.Printf("Payment method: %v\n", method)
	}
}

// Example 4: Configuration File Validation
func configurationExample() {
	fmt.Println("\n\n4. Configuration File Validation")
	fmt.Println("--------------------------------")

	// Custom validator: Connection string validation
	connectionStringValidator := func(value interface{}) error {
		connStr, _ := value.(string)
		
		// Parse connection string
		if strings.HasPrefix(connStr, "postgres://") {
			// PostgreSQL format: postgres://user:password@host:port/database
			postgresPattern := `^postgres://[^:]+:[^@]+@[^:]+:\d+/\w+$`
			if matched, _ := regexp.MatchString(postgresPattern, connStr); !matched {
				return fmt.Errorf("invalid PostgreSQL connection string format")
			}
		} else if strings.HasPrefix(connStr, "mongodb://") {
			// MongoDB format
			mongoPattern := `^mongodb://([^:]+:[^@]+@)?[^/]+/\w+`
			if matched, _ := regexp.MatchString(mongoPattern, connStr); !matched {
				return fmt.Errorf("invalid MongoDB connection string format")
			}
		} else {
			return fmt.Errorf("unsupported database type")
		}
		
		return nil
	}

	// Custom validator: CORS origins
	corsValidator := func(value interface{}) error {
		origins, ok := value.([]interface{})
		if !ok {
			return nil
		}
		
		for i, origin := range origins {
			originStr, _ := origin.(string)
			
			// Must be valid URL or wildcard
			if originStr != "*" {
				if !strings.HasPrefix(originStr, "http://") && 
				   !strings.HasPrefix(originStr, "https://") {
					return fmt.Errorf("origin %d must start with http:// or https://", i+1)
				}
				
				// No trailing slash
				if strings.HasSuffix(originStr, "/") {
					return fmt.Errorf("origin %d should not have trailing slash", i+1)
				}
			}
		}
		
		return nil
	}

	// Custom validator: Environment-specific rules
	envValidator := func(value interface{}) error {
		config, ok := value.(map[string]interface{})
		if !ok {
			return nil
		}
		
		env, _ := config["environment"].(string)
		server, _ := config["server"].(map[string]interface{})
		db, _ := config["database"].(map[string]interface{})
		
		// Handle port as either int or float64
		var port float64
		if server != nil {
			switch p := server["port"].(type) {
			case float64:
				port = p
			case int:
				port = float64(p)
			}
		}
		
		// Handle poolSize
		var poolSize float64
		if db != nil {
			switch p := db["maxConnections"].(type) {
			case float64:
				poolSize = p
			case int:
				poolSize = float64(p)
			}
		}
		
		switch env {
		case "development":
			// Dev constraints
			if port > 0 && (port < 3000 || port > 9999) {
				return fmt.Errorf("development port should be between 3000-9999 (got %.0f)", port)
			}
			if poolSize > 10 {
				return fmt.Errorf("development connection pool too large")
			}
			
		case "production":
			// Production constraints
			if port != 80 && port != 443 && port != 8080 {
				return fmt.Errorf("production port must be 80, 443, or 8080 (got %.0f)", port)
			}
			if poolSize < 20 {
				return fmt.Errorf("production needs at least 20 connections")
			}
			
			// Check security settings
			security, _ := config["security"].(map[string]interface{})
			if security == nil {
				return fmt.Errorf("security configuration required for production")
			}
			
			https, _ := security["forceHTTPS"].(bool)
			if !https {
				return fmt.Errorf("HTTPS must be enforced in production")
			}
			
			// Check CORS settings for production
			corsOrigins, _ := security["corsOrigins"].([]interface{})
			for _, origin := range corsOrigins {
				if origin == "*" {
					return fmt.Errorf("wildcard '*' not allowed in production CORS settings")
				}
			}
		}
		
		return nil
	}

	// Build configuration schema
	configSchema := builders.Object().
		Field("environment", builders.String().
			Enum("development", "staging", "production").
			Required()).
		Field("server", builders.Object().
			Field("host", builders.String().Required()).
			Field("port", builders.Number().
				Min(1).
				Max(65535).
				Integer().
				Required()).
			Field("workers", builders.Number().
				Min(1).
				Max(16).
				Integer())).
		Field("database", builders.Object().
			Field("connectionString", builders.String().
				Custom(connectionStringValidator).
				Required()).
			Field("maxConnections", builders.Number().
				Min(1).
				Max(100).
				Integer()).
			Field("timeout", builders.Number().Min(1000))).
		Field("cache", builders.Object().
			Field("provider", builders.String().
				Enum("memory", "redis", "memcached")).
			Field("ttl", builders.Number().Min(0)).
			Field("maxSize", builders.String().
				Pattern(`^\d+[KMG]B$`).
				Custom(func(value interface{}) error {
					size, _ := value.(string)
					// Parse size like "100MB", "2GB"
					if strings.HasSuffix(size, "KB") {
						return nil
					} else if strings.HasSuffix(size, "MB") {
						num := strings.TrimSuffix(size, "MB")
						if n, _ := fmt.Sscanf(num, "%d", new(int)); n != 1 {
							return fmt.Errorf("invalid size format")
						}
					} else if strings.HasSuffix(size, "GB") {
						num := strings.TrimSuffix(size, "GB")
						val, _ := fmt.Sscanf(num, "%d", new(int))
						if val > 16 {
							return fmt.Errorf("cache size exceeds 16GB limit")
						}
					}
					return nil
				}))).
		Field("security", builders.Object().
			Field("forceHTTPS", builders.Bool()).
			Field("corsOrigins", builders.Array().
				Of(builders.String()).
				Custom(corsValidator)).
			Field("rateLimit", builders.Object().
				Field("enabled", builders.Bool()).
				Field("requests", builders.Number().Min(1)).
				Field("window", builders.String().
					Pattern(`^\d+[smh]$`)))).
		Field("features", builders.Object().
			Custom(func(value interface{}) error {
				// Feature flags must be boolean
				features, _ := value.(map[string]interface{})
				for name, val := range features {
					if _, ok := val.(bool); !ok {
						return fmt.Errorf("feature flag '%s' must be boolean", name)
					}
				}
				return nil
			}).Optional()).
		Custom(envValidator) // Configuration-level validation

	// Test configurations
	configs := []map[string]interface{}{
		{
			"environment": "production",
			"server": map[string]interface{}{
				"host":    "api.example.com",
				"port":    8080,
				"workers": 8,
			},
			"database": map[string]interface{}{
				"connectionString": "postgres://user:pass@db.example.com:5432/proddb",
				"maxConnections":   50,
				"timeout":          5000,
			},
			"cache": map[string]interface{}{
				"provider": "redis",
				"ttl":      3600,
				"maxSize":  "2GB",
			},
			"security": map[string]interface{}{
				"forceHTTPS": true,
				"corsOrigins": []interface{}{
					"https://app.example.com",
					"https://admin.example.com",
				},
				"rateLimit": map[string]interface{}{
					"enabled":  true,
					"requests": 100,
					"window":   "1m",
				},
			},
			"features": map[string]interface{}{
				"newUI":        true,
				"betaFeatures": false,
				"v2API":        true,
			},
		},
		{
			"environment": "development",
			"server": map[string]interface{}{
				"host": "localhost",
				"port": 3000,
			},
			"database": map[string]interface{}{
				"connectionString": "invalid-connection-string", // Will fail
				"maxConnections":   5,
			},
			"security": map[string]interface{}{
				"corsOrigins": []interface{}{
					"*", // Will fail in production
				},
			},
		},
	}

	for i, config := range configs {
		fmt.Printf("\nValidating configuration %d (%s):\n", i+1, config["environment"])
		if err := qf.ValidateWithMode(config, configSchema, qf.Loose); err != nil {
			fmt.Printf("[X] Configuration validation failed:\n%v\n", err)
		} else {
			fmt.Println("[✓] Configuration validation passed!")
		}
	}
}

// Helper Functions

func isBalancedUsername(username string) bool {
	stack := []rune{}
	pairs := map[rune]rune{
		')': '(',
		']': '[',
		'}': '{',
	}
	
	for _, ch := range username {
		switch ch {
		case '(', '[', '{':
			stack = append(stack, ch)
		case ')', ']', '}':
			if len(stack) == 0 || stack[len(stack)-1] != pairs[ch] {
				return false
			}
			stack = stack[:len(stack)-1]
		}
	}
	
	return len(stack) == 0
}

func calculatePasswordStrength(password string) int {
	strength := 0
	
	if len(password) >= 8 {
		strength++
	}
	if len(password) >= 12 {
		strength++
	}
	if regexp.MustCompile(`[a-z]`).MatchString(password) {
		strength++
	}
	if regexp.MustCompile(`[A-Z]`).MatchString(password) {
		strength++
	}
	if regexp.MustCompile(`[0-9]`).MatchString(password) {
		strength++
	}
	if regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password) {
		strength++
	}
	
	// Penalize repeated characters (3 or more of the same character)
	for i := 0; i < len(password)-2; i++ {
		if password[i] == password[i+1] && password[i+1] == password[i+2] {
			strength--
			break
		}
	}
	
	if strength < 0 {
		strength = 0
	} else if strength > 5 {
		strength = 5
	}
	
	return strength
}

func calculateAge(birthDate time.Time) int {
	now := time.Now()
	years := now.Year() - birthDate.Year()
	
	if now.YearDay() < birthDate.YearDay() {
		years--
	}
	
	return years
}

func phoneNumberValidator(value interface{}) error {
	phone, _ := value.(string)
	
	// Remove common formatting
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) || r == '+' {
			return r
		}
		return -1
	}, phone)
	
	// International format
	if strings.HasPrefix(cleaned, "+") {
		if len(cleaned) < 10 || len(cleaned) > 15 {
			return fmt.Errorf("invalid international phone number length")
		}
	} else {
		// Assume US/Canada
		if len(cleaned) != 10 {
			return fmt.Errorf("phone number must be 10 digits")
		}
	}
	
	return nil
}

func referralCodeValidator(value interface{}) error {
	code, _ := value.(string)
	
	if code == "" {
		return nil // Optional field
	}
	
	// Format: 4-10 alphanumeric characters
	pattern := `^[A-Z0-9]{4,10}$`
	if matched, _ := regexp.MatchString(pattern, code); !matched {
		return fmt.Errorf("invalid referral code format (must be 4-10 alphanumeric characters)")
	}
	
	// Check against blacklist
	blacklist := []string{"TEST", "ADMIN", "FREE", "HACK"}
	for _, blocked := range blacklist {
		if strings.Contains(code, blocked) {
			return fmt.Errorf("referral code contains blocked word")
		}
	}
	
	return nil
}

func isValidCreditCard(number string) bool {
	// Luhn algorithm
	if len(number) < 13 || len(number) > 19 {
		return false
	}
	
	sum := 0
	isEven := false
	
	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')
		
		if isEven {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		
		sum += digit
		isEven = !isEven
	}
	
	return sum%10 == 0
}

func ipAddressValidator(value interface{}) error {
	ip, _ := value.(string)
	
	// Parse IP
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address format")
	}
	
	// Check for reserved IPs
	if parsedIP.IsLoopback() {
		return fmt.Errorf("loopback IP not allowed")
	}
	
	if parsedIP.IsPrivate() {
		// You might want to allow this depending on your use case
		// return fmt.Errorf("private IP addresses not allowed")
	}
	
	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}