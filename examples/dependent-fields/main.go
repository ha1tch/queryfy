package main

import (
	"fmt"

	qf "github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
)

func main() {
	fmt.Println("=== Queryfy Dependent Fields Validation Examples ===")

	// Example 1: Account type dependent validation
	accountExample()

	// Example 2: Shipping address dependent validation
	shippingExample()

	// Example 3: Payment method dependent validation
	paymentExample()

	// Example 4: Complex multi-field dependencies
	complexDependencyExample()
}

func accountExample() {
	fmt.Println("1. Account Type Dependent Validation")
	fmt.Println("------------------------------------")

	// Schema with dependent fields based on account type
	accountSchema := builders.Object().WithDependencies().
		Field("accountType", builders.String().
			Enum("personal", "business", "nonprofit").
			Required()).
		Field("firstName", builders.String().Required()).
		Field("lastName", builders.String().Required()).
		// Company name required for business accounts
		DependentField("companyName",
			builders.Dependent("companyName").
				On("accountType").
				When(builders.WhenEquals("accountType", "business")).
				Then(builders.String().MinLength(2).Required()).
				Else(builders.String().Optional())).
		// Tax ID required for business and nonprofit
		DependentField("taxId",
			builders.Dependent("taxId").
				On("accountType").
				When(builders.WhenIn("accountType", "business", "nonprofit")).
				Then(builders.String().Pattern(`^\d{2}-\d{7}$`).Required())).
		// Annual revenue required only for business
		DependentField("annualRevenue",
			builders.Dependent("annualRevenue").
				On("accountType").
				When(builders.WhenEquals("accountType", "business")).
				Then(builders.Number().Min(0).Required())).
		// Nonprofit status only for nonprofit accounts
		DependentField("taxExemptStatus",
			builders.Dependent("taxExemptStatus").
				On("accountType").
				When(builders.WhenEquals("accountType", "nonprofit")).
				Then(builders.String().Enum("501c3", "501c4", "other").Required()))

	testAccounts := []map[string]interface{}{
		{
			"accountType": "personal",
			"firstName":   "John",
			"lastName":    "Doe",
			// No company fields needed
		},
		{
			"accountType":   "business",
			"firstName":     "Jane",
			"lastName":      "Smith",
			"companyName":   "Acme Corp",
			"taxId":         "12-3456789",
			"annualRevenue": 1000000,
		},
		{
			"accountType":     "nonprofit",
			"firstName":       "Bob",
			"lastName":        "Johnson",
			"companyName":     "Helping Hands Foundation",
			"taxId":           "98-7654321",
			"taxExemptStatus": "501c3",
		},
		{
			"accountType": "business",
			"firstName":   "Alice",
			"lastName":    "Brown",
			// Missing required business fields
		},
	}

	for i, account := range testAccounts {
		fmt.Printf("\nValidating account %d (%s):\n", i+1, account["accountType"])
		if err := qf.Validate(account, accountSchema); err != nil {
			fmt.Printf("[X] Validation failed:\n%v\n", err)
		} else {
			fmt.Printf("[✓] Account is valid!\n")
		}
	}
}

func shippingExample() {
	fmt.Println("\n\n2. Shipping Address Dependent Validation")
	fmt.Println("----------------------------------------")

	// Schema where shipping address fields depend on shipping method
	orderSchema := builders.Object().WithDependencies().
		Field("items", builders.Array().MinItems(1).Required()).
		Field("shippingMethod", builders.String().
			Enum("pickup", "standard", "express", "international").
			Required()).
		// Shipping address required except for pickup
		DependentField("shippingAddress",
			builders.Dependent("shippingAddress").
				On("shippingMethod").
				When(builders.WhenNotEquals("shippingMethod", "pickup")).
				Then(builders.Object().
					Field("street", builders.String().Required()).
					Field("city", builders.String().Required()).
					Field("state", builders.String().Required()).
					Field("country", builders.String().Required()).
					Required())).
		// Postal code format depends on country
		DependentField("postalCode",
			builders.Dependent("postalCode").
				On("shippingMethod", "shippingAddress").
				When(builders.WhenAll(
					builders.WhenNotEquals("shippingMethod", "pickup"),
					builders.WhenExists("shippingAddress"),
				)).
				Then(builders.String().Required())).
		// Express shipping requires phone number
		DependentField("contactPhone",
			builders.Dependent("contactPhone").
				On("shippingMethod").
				When(builders.WhenIn("shippingMethod", "express", "international")).
				Then(builders.String().Pattern(`^\+?[\d\s-()]+$`).Required())).
		// International shipping requires customs info
		DependentField("customsDeclaration",
			builders.Dependent("customsDeclaration").
				On("shippingMethod").
				When(builders.WhenEquals("shippingMethod", "international")).
				Then(builders.Object().
					Field("description", builders.String().Required()).
					Field("value", builders.Number().Min(0).Required()).
					Field("hsCode", builders.String().Pattern(`^\d{6,10}$`).Required()).
					Required()))

	testOrders := []map[string]interface{}{
		{
			"items":          []interface{}{"item1", "item2"},
			"shippingMethod": "pickup",
			// No address needed for pickup
		},
		{
			"items":          []interface{}{"item3"},
			"shippingMethod": "standard",
			"shippingAddress": map[string]interface{}{
				"street":  "123 Main St",
				"city":    "New York",
				"state":   "NY",
				"country": "US",
			},
			"postalCode": "10001",
		},
		{
			"items":          []interface{}{"item4"},
			"shippingMethod": "express",
			"shippingAddress": map[string]interface{}{
				"street":  "456 Oak Ave",
				"city":    "Los Angeles",
				"state":   "CA",
				"country": "US",
			},
			"postalCode":   "90001",
			"contactPhone": "+1-555-123-4567",
		},
		{
			"items":          []interface{}{"item5"},
			"shippingMethod": "international",
			"shippingAddress": map[string]interface{}{
				"street":  "789 Maple Rd",
				"city":    "Toronto",
				"state":   "ON",
				"country": "CA",
			},
			"postalCode":   "M5H 2N2",
			"contactPhone": "+1-416-555-0123",
			"customsDeclaration": map[string]interface{}{
				"description": "Electronic accessories",
				"value":       150.00,
				"hsCode":      "851770",
			},
		},
		{
			"items":          []interface{}{"item6"},
			"shippingMethod": "express",
			// Missing required fields for express shipping
		},
	}

	for i, order := range testOrders {
		fmt.Printf("\nValidating order %d (%s shipping):\n", i+1, order["shippingMethod"])
		if err := qf.Validate(order, orderSchema); err != nil {
			fmt.Printf("[X] Validation failed:\n%v\n", err)
		} else {
			fmt.Printf("[✓] Order is valid!\n")
		}
	}
}

func paymentExample() {
	fmt.Println("\n\n3. Payment Method Dependent Validation")
	fmt.Println("--------------------------------------")

	// Different fields required based on payment method
	paymentSchema := builders.Object().WithDependencies().
		Field("amount", builders.Number().Min(0.01).Required()).
		Field("currency", builders.String().Length(3).Required()).
		Field("paymentMethod", builders.String().
			Enum("credit_card", "bank_transfer", "paypal", "crypto").
			Required()).
		// Credit card fields
		DependentField("cardNumber",
			builders.Dependent("cardNumber").
				When(builders.WhenEquals("paymentMethod", "credit_card")).
				Then(builders.String().Pattern(`^\d{13,19}$`).Required())).
		DependentField("cardExpiry",
			builders.Dependent("cardExpiry").
				When(builders.WhenEquals("paymentMethod", "credit_card")).
				Then(builders.String().Pattern(`^(0[1-9]|1[0-2])\/\d{2}$`).Required())).
		DependentField("cvv",
			builders.Dependent("cvv").
				When(builders.WhenEquals("paymentMethod", "credit_card")).
				Then(builders.String().Pattern(`^\d{3,4}$`).Required())).
		// Bank transfer fields
		DependentField("accountNumber",
			builders.Dependent("accountNumber").
				When(builders.WhenEquals("paymentMethod", "bank_transfer")).
				Then(builders.String().Required())).
		DependentField("routingNumber",
			builders.Dependent("routingNumber").
				When(builders.WhenEquals("paymentMethod", "bank_transfer")).
				Then(builders.String().Pattern(`^\d{9}$`).Required())).
		// PayPal fields
		DependentField("paypalEmail",
			builders.Dependent("paypalEmail").
				When(builders.WhenEquals("paymentMethod", "paypal")).
				Then(builders.String().Email().Required())).
		// Crypto fields
		DependentField("walletAddress",
			builders.Dependent("walletAddress").
				When(builders.WhenEquals("paymentMethod", "crypto")).
				Then(builders.String().MinLength(26).Required())).
		DependentField("cryptoCurrency",
			builders.Dependent("cryptoCurrency").
				When(builders.WhenEquals("paymentMethod", "crypto")).
				Then(builders.String().Enum("BTC", "ETH", "USDT").Required())).
		// High value transactions require additional verification
		DependentField("verificationCode",
			builders.Dependent("verificationCode").
				On("amount", "paymentMethod").
				When(builders.WhenAll(
					builders.WhenGreaterThan("amount", 10000),
					builders.WhenNotEquals("paymentMethod", "credit_card"), // CC has its own verification
				)).
				Then(builders.String().Length(6).Required()))

	testPayments := []map[string]interface{}{
		{
			"amount":        99.99,
			"currency":      "USD",
			"paymentMethod": "credit_card",
			"cardNumber":    "4111111111111111",
			"cardExpiry":    "12/25",
			"cvv":           "123",
		},
		{
			"amount":        500.00,
			"currency":      "USD",
			"paymentMethod": "bank_transfer",
			"accountNumber": "123456789",
			"routingNumber": "021000021",
		},
		{
			"amount":        50.00,
			"currency":      "EUR",
			"paymentMethod": "paypal",
			"paypalEmail":   "user@example.com",
		},
		{
			"amount":         0.05,
			"currency":       "BTC",
			"paymentMethod":  "crypto",
			"walletAddress":  "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
			"cryptoCurrency": "BTC",
		},
		{
			"amount":           15000.00,
			"currency":         "USD",
			"paymentMethod":    "bank_transfer",
			"accountNumber":    "987654321",
			"routingNumber":    "021000021",
			"verificationCode": "ABC123", // Required for high value
		},
		{
			"amount":        100.00,
			"currency":      "USD",
			"paymentMethod": "credit_card",
			// Missing required credit card fields
		},
	}

	for i, payment := range testPayments {
		fmt.Printf("\nValidating payment %d (%.2f %s via %s):\n",
			i+1, payment["amount"], payment["currency"], payment["paymentMethod"])
		if err := qf.Validate(payment, paymentSchema); err != nil {
			fmt.Printf("[X] Validation failed:\n%v\n", err)
		} else {
			fmt.Printf("[✓] Payment is valid!\n")
		}
	}
}

func complexDependencyExample() {
	fmt.Println("\n\n4. Complex Multi-Field Dependencies")
	fmt.Println("-----------------------------------")

	// Insurance form with complex interdependencies
	insuranceSchema := builders.Object().WithDependencies().
		Field("insuranceType", builders.String().
			Enum("auto", "home", "life", "health").
			Required()).
		Field("primaryHolder", builders.Object().
			Field("name", builders.String().Required()).
			Field("age", builders.Number().Min(0).Max(150).Required()).
			Field("smoker", builders.Bool())).
		// Auto insurance specific fields
		DependentField("vehicle",
			builders.Dependent("vehicle").
				When(builders.WhenEquals("insuranceType", "auto")).
				Then(builders.Object().
					Field("make", builders.String().Required()).
					Field("model", builders.String().Required()).
					Field("year", builders.Number().Min(1900).Required()).
					Field("mileage", builders.Number().Min(0).Required()).
					Required())).
		// Home insurance specific fields
		DependentField("property",
			builders.Dependent("property").
				When(builders.WhenEquals("insuranceType", "home")).
				Then(builders.Object().
					Field("address", builders.String().Required()).
					Field("sqft", builders.Number().Min(100).Required()).
					Field("yearBuilt", builders.Number().Min(1800).Required()).
					Field("hasAlarm", builders.Bool().Required()).
					Required())).
		// Life insurance requires beneficiary
		DependentField("beneficiary",
			builders.Dependent("beneficiary").
				When(builders.WhenEquals("insuranceType", "life")).
				Then(builders.Object().
					Field("name", builders.String().Required()).
					Field("relationship", builders.String().Required()).
					Field("percentage", builders.Number().Min(0).Max(100).Required()).
					Required())).
		// Health insurance - smoker surcharge info required if smoker
		DependentField("smokerDetails",
			builders.Dependent("smokerDetails").
				On("insuranceType", "primaryHolder").
				When(func(data map[string]interface{}) bool {
					if data["insuranceType"] != "health" {
						return false
					}
					if holder, ok := data["primaryHolder"].(map[string]interface{}); ok {
						if smoker, ok := holder["smoker"].(bool); ok {
							return smoker
						}
					}
					return false
				}).
				Then(builders.Object().
					Field("yearsSmoked", builders.Number().Min(0).Required()).
					Field("packsPerDay", builders.Number().Min(0).Required()).
					Required())).
		// Premium calculation factors - depends on multiple fields
		DependentField("premiumFactors",
			builders.Dependent("premiumFactors").
				When(func(data map[string]interface{}) bool {
					// Always calculate premium factors
					return true
				}).
				Then(builders.Object().
					Field("basePremium", builders.Number().Min(0).Required()).
					Field("riskMultiplier", builders.Number().Min(0.5).Max(5.0)).
					Custom(func(value interface{}) error {
						// Custom validation based on insurance type and other factors
						return nil
					})))

	testApplications := []map[string]interface{}{
		{
			"insuranceType": "auto",
			"primaryHolder": map[string]interface{}{
				"name": "John Doe",
				"age":  35,
			},
			"vehicle": map[string]interface{}{
				"make":    "Toyota",
				"model":   "Camry",
				"year":    2020,
				"mileage": 25000,
			},
			"premiumFactors": map[string]interface{}{
				"basePremium":    1200,
				"riskMultiplier": 1.0,
			},
		},
		{
			"insuranceType": "home",
			"primaryHolder": map[string]interface{}{
				"name": "Jane Smith",
				"age":  42,
			},
			"property": map[string]interface{}{
				"address":   "123 Oak Street",
				"sqft":      2500,
				"yearBuilt": 1995,
				"hasAlarm":  true,
			},
			"premiumFactors": map[string]interface{}{
				"basePremium":    800,
				"riskMultiplier": 0.9, // Discount for alarm
			},
		},
		{
			"insuranceType": "life",
			"primaryHolder": map[string]interface{}{
				"name":   "Bob Johnson",
				"age":    50,
				"smoker": false,
			},
			"beneficiary": map[string]interface{}{
				"name":         "Alice Johnson",
				"relationship": "spouse",
				"percentage":   100,
			},
			"premiumFactors": map[string]interface{}{
				"basePremium":    500,
				"riskMultiplier": 1.2,
			},
		},
		{
			"insuranceType": "health",
			"primaryHolder": map[string]interface{}{
				"name":   "Charlie Brown",
				"age":    45,
				"smoker": true,
			},
			"smokerDetails": map[string]interface{}{
				"yearsSmoked": 20,
				"packsPerDay": 1.5,
			},
			"premiumFactors": map[string]interface{}{
				"basePremium":    400,
				"riskMultiplier": 2.5, // High due to smoking
			},
		},
		{
			"insuranceType": "auto",
			"primaryHolder": map[string]interface{}{
				"name": "Invalid Application",
				"age":  25,
			},
			// Missing required vehicle information
			"premiumFactors": map[string]interface{}{
				"basePremium": 1000,
			},
		},
	}

	for i, app := range testApplications {
		fmt.Printf("\nValidating insurance application %d (%s):\n", i+1, app["insuranceType"])
		if err := qf.Validate(app, insuranceSchema); err != nil {
			fmt.Printf("[X] Validation failed:\n%v\n", err)
		} else {
			fmt.Printf("[✓] Application is valid!\n")
		}
	}

	// Example of using custom dependency conditions
	fmt.Println("\n\n5. Custom Dependency Conditions")
	fmt.Println("-------------------------------")

	// Discount eligibility based on complex conditions
	discountSchema := builders.Object().WithDependencies().
		Field("customerType", builders.String().Enum("new", "returning", "vip")).
		Field("orderTotal", builders.Number().Min(0)).
		Field("itemCount", builders.Number().Min(0).Integer()).
		DependentField("discountCode",
			builders.Dependent("discountCode").
				When(builders.WhenAny(
					// VIP always gets discount
					builders.WhenEquals("customerType", "vip"),
					// Or high value orders
					builders.WhenGreaterThan("orderTotal", 500),
					// Or bulk orders
					builders.WhenGreaterThan("itemCount", 20),
					// Or returning customer with decent order
					builders.WhenAll(
						builders.WhenEquals("customerType", "returning"),
						builders.WhenGreaterThan("orderTotal", 100),
					),
				)).
				Then(builders.String().Pattern(`^[A-Z0-9]{6,10}$`).Required()).
				Else(builders.String().Optional()))

	discountTests := []map[string]interface{}{
		{
			"customerType": "vip",
			"orderTotal":   50.0,
			"itemCount":    2.0,
			"discountCode": "VIP2024", // Required for VIP
		},
		{
			"customerType": "new",
			"orderTotal":   600.0,
			"itemCount":    5.0,
			"discountCode": "BULK500", // Required for high value
		},
		{
			"customerType": "returning",
			"orderTotal":   150.0,
			"itemCount":    3.0,
			"discountCode": "RETURN20", // Required for returning + >$100
		},
		{
			"customerType": "new",
			"orderTotal":   75.0,
			"itemCount":    2.0,
			// No discount code needed
		},
	}

	for i, order := range discountTests {
		fmt.Printf("\nValidating discount eligibility %d:\n", i+1)
		fmt.Printf("  Customer: %s, Total: $%.2f, Items: %.0f\n",
			order["customerType"], order["orderTotal"], order["itemCount"])
		if err := qf.Validate(order, discountSchema); err != nil {
			fmt.Printf("[X] Validation failed:\n%v\n", err)
		} else {
			fmt.Printf("[✓] Order is valid!\n")
		}
	}
}
