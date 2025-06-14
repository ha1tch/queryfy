// phone.go - Phone number normalization for multiple countries
package transformers

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/ha1tch/queryfy/builders"
)

// PhoneFormat defines the format for a country's phone numbers.
type PhoneFormat struct {
	CountryCode   string
	CountryPrefix string
	Formats       []string // Possible input formats (regex)
	NormalizedLen int      // Expected length after normalization (excluding country code)
	OutputFormat  string   // How to format the normalized number
}

// phoneFormats contains phone formats for all supported countries.
var phoneFormats = map[string]PhoneFormat{
	// North America
	"US": {
		CountryCode:   "US",
		CountryPrefix: "+1",
		Formats: []string{
			`^\+?1?[-.\s]?\(?(\d{3})\)?[-.\s]?(\d{3})[-.\s]?(\d{4})$`,
			`^(\d{3})[-.\s]?(\d{3})[-.\s]?(\d{4})$`,
		},
		NormalizedLen: 10,
		OutputFormat:  "+1AAABBBCCCC",
	},

	// South America
	"AR": { // Argentina
		CountryCode:   "AR",
		CountryPrefix: "+54",
		Formats: []string{
			// International format with 9 and spaces
			`^\+?54[-.\s]?9[-.\s]?(\d{2,4})[-.\s]?(\d{4})[-.\s]?(\d{4})$`,
			`^\+?54[-.\s]?9[-.\s]?(\d{2})[-.\s]?(\d{4})[-.\s]?(\d{4})$`,
			`^\+?54[-.\s]?9[-.\s]?(\d{2,4})[-.\s]?(\d{6,8})$`,
			// Specific format for "+54 9 11 1234 5678"
			`^\+?54\s+9\s+(\d{2})\s+(\d{4})\s+(\d{4})$`,
			// Local mobile with 15 (Buenos Aires and others)
			`^0?(\d{2,4})[-.\s]?15[-.\s]?(\d{3,4})[-.\s]?(\d{3,4})$`,
			// Alternative mobile format
			`^(\d{2,4})[-.\s]?15[-.\s]?(\d{3,4})[-.\s]?(\d{3,4})$`,
			// Without 15 prefix
			`^15[-.\s]?(\d{3,4})[-.\s]?(\d{3,4})$`,
			// Compact formats
			`^0?(\d{2,4})15(\d{6,8})$`,
			`^(\d{10,11})$`,
		},
		NormalizedLen: 10,
		OutputFormat:  "+54 9 AAA BBBBBBBB",
	},
	"BR": { // Brazil
		CountryCode:   "BR",
		CountryPrefix: "+55",
		Formats: []string{
			`^\+?55?[-.\s]?\(?0?(\d{2})\)?[-.\s]?9?(\d{4,5})[-.\s]?(\d{4})$`,
			`^\(?0?(\d{2})\)?[-.\s]?9?(\d{4,5})[-.\s]?(\d{4})$`,
			// Carrier code format
			`^0(\d{2})[-.\s]?9?(\d{4,5})[-.\s]?(\d{4})$`,
		},
		NormalizedLen: 11,
		OutputFormat:  "+55 AA 9NNNN-NNNN",
	},
	"BO": { // Bolivia
		CountryCode:   "BO",
		CountryPrefix: "+591",
		Formats: []string{
			`^\+?591?[-.\s]?([67]\d{7})$`,
			`^([67]\d{7})$`,
			`^\+?591?[-.\s]?([2-4]\d{6})$`,
		},
		NormalizedLen: 8,
		OutputFormat:  "+591 NNNNNNNN",
	},
	"CL": { // Chile
		CountryCode:   "CL",
		CountryPrefix: "+56",
		Formats: []string{
			// International formats
			`^\+?56[-.\s]?9[-.\s]?(\d{4})[-.\s]?(\d{4})$`,
			`^\+?56[-.\s]?9(\d{8})$`,
			// Local formats with spaces
			`^0?9[-.\s]?(\d{4})[-.\s]?(\d{4})$`,
			`^9[-.\s]?(\d{4})[-.\s]?(\d{4})$`,
			// Compact format
			`^0?9(\d{8})$`,
			`^9(\d{8})$`,
		},
		NormalizedLen: 9,
		OutputFormat:  "+56 9 NNNNNNNN",
	},
	"CO": { // Colombia
		CountryCode:   "CO",
		CountryPrefix: "+57",
		Formats: []string{
			// International formats
			`^\+?57[-.\s]?3(\d{2})[-.\s]?(\d{3})[-.\s]?(\d{4})$`,
			`^\+?57[-.\s]?3(\d{9})$`,
			// Local formats with spaces
			`^3(\d{2})[-.\s]?(\d{3})[-.\s]?(\d{4})$`,
			// Compact format
			`^3(\d{9})$`,
		},
		NormalizedLen: 10,
		OutputFormat:  "+57 3NNNNNNNNN",
	},
	"EC": { // Ecuador
		CountryCode:   "EC",
		CountryPrefix: "+593",
		Formats: []string{
			`^\+?593?[-.\s]?9(\d{8})$`,
			`^0?9(\d{8})$`,
			`^\+?593?[-.\s]?(\d{8,9})$`,
		},
		NormalizedLen: 9,
		OutputFormat:  "+593 9NNNNNNNN",
	},
	"GF": { // French Guiana
		CountryCode:   "GF",
		CountryPrefix: "+594",
		Formats: []string{
			`^\+?594?[-.\s]?694(\d{6})$`,
			`^0?694(\d{6})$`,
			`^\+?594?[-.\s]?594(\d{6})$`,
		},
		NormalizedLen: 9,
		OutputFormat:  "+594 694NNNNNN",
	},
	"GY": { // Guyana
		CountryCode:   "GY",
		CountryPrefix: "+592",
		Formats: []string{
			`^\+?592?[-.\s]?6(\d{6})$`,
			`^6(\d{6})$`,
			`^\+?592?[-.\s]?(\d{7})$`,
		},
		NormalizedLen: 7,
		OutputFormat:  "+592 NNNNNNN",
	},
	"PY": { // Paraguay
		CountryCode:   "PY",
		CountryPrefix: "+595",
		Formats: []string{
			`^\+?595?[-.\s]?9(\d{8})$`,
			`^0?9(\d{8})$`,
			`^\+?595?[-.\s]?(\d{8,9})$`,
		},
		NormalizedLen: 9,
		OutputFormat:  "+595 9NNNNNNNN",
	},
	"PE": { // Peru
		CountryCode:   "PE",
		CountryPrefix: "+51",
		Formats: []string{
			`^\+?51?[-.\s]?9(\d{8})$`,
			`^9(\d{8})$`,
			`^\+?51?[-.\s]?(\d{9})$`,
		},
		NormalizedLen: 9,
		OutputFormat:  "+51 9NNNNNNNN",
	},
	"SR": { // Suriname
		CountryCode:   "SR",
		CountryPrefix: "+597",
		Formats: []string{
			`^\+?597?[-.\s]?[78](\d{6})$`,
			`^[78](\d{6})$`,
			`^\+?597?[-.\s]?(\d{6,7})$`,
		},
		NormalizedLen: 7,
		OutputFormat:  "+597 NNNNNNN",
	},
	"UY": { // Uruguay
		CountryCode:   "UY",
		CountryPrefix: "+598",
		Formats: []string{
			// International formats
			`^\+?598[-.\s]?9?(\d{1})[-.\s]?(\d{3})[-.\s]?(\d{3})$`,
			`^\+?598[-.\s]?0?9(\d{7})$`,
			// Local formats with spaces
			`^0?9(\d{1})[-.\s]?(\d{3})[-.\s]?(\d{3})$`,
			`^9(\d{1})[-.\s]?(\d{3})[-.\s]?(\d{3})$`,
			// Compact formats
			`^0?9(\d{7})$`,
			`^0?9(\d{8})$`,
		},
		NormalizedLen: 8,
		OutputFormat:  "+598 9NNNNNNN",
	},
	"VE": { // Venezuela
		CountryCode:   "VE",
		CountryPrefix: "+58",
		Formats: []string{
			`^\+?58?[-.\s]?4(\d{9})$`,
			`^0?4(\d{9})$`,
			`^\+?58?[-.\s]?(\d{10})$`,
		},
		NormalizedLen: 10,
		OutputFormat:  "+58 4NNNNNNNNN",
	},

	// Europe
	"UK": { // United Kingdom
		CountryCode:   "UK",
		CountryPrefix: "+44",
		Formats: []string{
			`^\+?44?[-.\s]?0?7(\d{9})$`,
			`^0?7(\d{9})$`,
			`^\+?44?[-.\s]?\(?0?(\d{3,4})\)?[-.\s]?(\d{6,7})$`,
		},
		NormalizedLen: 10,
		OutputFormat:  "+44 7NNNNNNNNN",
	},

	// Central America
	"MX": { // Mexico
		CountryCode:   "MX",
		CountryPrefix: "+52",
		Formats: []string{
			`^\+?52?[-.\s]?1?[-.\s]?(\d{3})[-.\s]?(\d{3})[-.\s]?(\d{4})$`,
			`^(\d{3})[-.\s]?(\d{3})[-.\s]?(\d{4})$`,
		},
		NormalizedLen: 10,
		OutputFormat:  "+52 1 AAA BBB CCCC",
	},
}

// NormalizePhone creates a transformer for phone normalization.
func NormalizePhone(defaultCountry string) builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}

		// First, try to normalize with the raw input (handles formatted numbers)
		normalized, err := normalizeForCountry(str, defaultCountry)
		if err == nil {
			return normalized, nil
		}

		// If that fails, clean the input and detect country
		cleaned := strings.Map(func(r rune) rune {
			if unicode.IsDigit(r) || r == '+' {
				return r
			}
			return -1
		}, str)

		// Try to detect country by prefix
		country := detectCountry(cleaned, defaultCountry)

		// Try again with detected country
		normalized, err = normalizeForCountry(str, country)
		if err != nil {
			return nil, err
		}

		return normalized, nil
	}
}

// detectCountry detects the country from phone number prefix.
func detectCountry(phone string, defaultCountry string) string {
	// Extended country prefix map for all South America
	countryPrefixes := map[string]string{
		"+1":   "US",
		"+44":  "UK",
		"+52":  "MX",
		"+54":  "AR",
		"+55":  "BR",
		"+56":  "CL",
		"+57":  "CO",
		"+51":  "PE",
		"+58":  "VE",
		"+591": "BO",
		"+592": "GY",
		"+593": "EC",
		"+594": "GF",
		"+595": "PY",
		"+597": "SR",
		"+598": "UY",
	}

	// Check explicit country codes
	for prefix, country := range countryPrefixes {
		if strings.HasPrefix(phone, prefix) {
			return country
		}
	}

	// Special mobile number patterns for South America
	if len(phone) > 0 {
		// Argentina mobile prefix
		if defaultCountry == "AR" && strings.HasPrefix(phone, "15") {
			return "AR"
		}

		// Common mobile prefixes
		mobileFirstDigit := string(phone[0])
		switch defaultCountry {
		case "BO":
			if mobileFirstDigit == "6" || mobileFirstDigit == "7" {
				return "BO"
			}
		case "VE":
			if mobileFirstDigit == "4" {
				return "VE"
			}
		case "CO":
			if mobileFirstDigit == "3" {
				return "CO"
			}
		}

		// Many SA countries use 9 as mobile prefix
		if mobileFirstDigit == "9" {
			switch defaultCountry {
			case "BR", "CL", "EC", "PY", "PE", "UY":
				return defaultCountry
			}
		}
	}

	// Brazilian area code detection
	if defaultCountry == "BR" && len(phone) >= 2 {
		brazilAreaCodes := map[string]bool{
			"11": true, "12": true, "13": true, "14": true, "15": true,
			"16": true, "17": true, "18": true, "19": true,
			"21": true, "22": true, "24": true, "27": true, "28": true,
			"31": true, "32": true, "33": true, "34": true, "35": true,
			"37": true, "38": true, "41": true, "42": true, "43": true,
			"44": true, "45": true, "46": true, "47": true, "48": true,
			"49": true, "51": true, "53": true, "54": true, "55": true,
			"61": true, "62": true, "63": true, "64": true, "65": true,
			"66": true, "67": true, "68": true, "69": true, "71": true,
			"73": true, "74": true, "75": true, "77": true, "79": true,
			"81": true, "82": true, "83": true, "84": true, "85": true,
			"86": true, "87": true, "88": true, "89": true, "91": true,
			"92": true, "93": true, "94": true, "95": true, "96": true,
			"97": true, "98": true, "99": true,
		}

		areaCode := phone[:2]
		if brazilAreaCodes[areaCode] {
			return "BR"
		}
	}

	return defaultCountry
}

// normalizeForCountry normalizes a phone number for a specific country.
func normalizeForCountry(phone string, country string) (string, error) {
	format, exists := phoneFormats[strings.ToUpper(country)]
	if !exists {
		return "", fmt.Errorf("unsupported country code: %s", country)
	}

	// Clean the phone for digit extraction later
	cleaned := strings.TrimSpace(phone)

	// Try each format pattern
	for _, pattern := range format.Formats {
		re := regexp.MustCompile(pattern)
		if re.MatchString(cleaned) {
			// Special handling for Argentina international format
			if country == "AR" && strings.HasPrefix(cleaned, "+54 9 ") {
				// For "+54 9 11 1234 5678" format, extract the digits properly
				digits := strings.Map(func(r rune) rune {
					if unicode.IsDigit(r) {
						return r
					}
					return -1
				}, cleaned)
				// Remove country code "54"
				digits = strings.TrimPrefix(digits, "54")
				// We should have "9111234567" (11 digits with the 9)
				if len(digits) == 11 && strings.HasPrefix(digits, "9") {
					// Return with proper format: +54 9 followed by 10 digits
					return format.CountryPrefix + digits, nil
				}
			}

			// Extract digits only
			digits := strings.Map(func(r rune) rune {
				if unicode.IsDigit(r) {
					return r
				}
				return -1
			}, cleaned)

			// Remove country code if present
			countryDigits := strings.TrimPrefix(format.CountryPrefix, "+")
			if strings.HasPrefix(digits, countryDigits) {
				digits = strings.TrimPrefix(digits, countryDigits)
			}

			// Handle special cases for normalization
			switch country {
			case "AR":
				// Remove leading 0 if present
				digits = strings.TrimPrefix(digits, "0")
				// Remove 15 if it's in the middle
				if len(digits) > 10 {
					// Try to find and remove "15" after area code
					if idx := strings.Index(digits[2:], "15"); idx >= 0 {
						digits = digits[:idx+2] + digits[idx+4:]
					}
				}
				// Ensure we have exactly 10 digits
				if len(digits) == 10 {
					return format.CountryPrefix + "9" + digits, nil
				}
			case "BR":
				// Remove leading 0 if present
				digits = strings.TrimPrefix(digits, "0")
				// Ensure 9 for mobile numbers
				if len(digits) == 10 {
					// Add 9 after area code if not present
					digits = digits[:2] + "9" + digits[2:]
				}
				if len(digits) == 11 {
					return format.CountryPrefix + digits, nil
				}
			case "CL":
				// Remove leading 0 if present
				digits = strings.TrimPrefix(digits, "0")
				if len(digits) == 9 {
					return format.CountryPrefix + digits, nil
				}
			case "CO":
				if len(digits) == 10 {
					return format.CountryPrefix + digits, nil
				}
			case "UY":
				// Remove leading 0 if present
				digits = strings.TrimPrefix(digits, "0")
				// Uruguay mobile numbers are 8 digits total (including the 9)
				// Format: 9X XXX XXX where 9 is the mobile prefix
				if len(digits) == 8 && strings.HasPrefix(digits, "9") {
					// Already has the 9 prefix, use as is
					return format.CountryPrefix + digits, nil
				} else if len(digits) == 7 {
					// Missing the 9 prefix, add it
					return format.CountryPrefix + "9" + digits, nil
				} else if len(digits) == 9 && strings.HasPrefix(digits, "99") {
					// Has double 9 (like 099 -> 99), remove one
					return format.CountryPrefix + digits[1:], nil
				}
			default:
				// For other countries, just ensure proper length
				if len(digits) >= format.NormalizedLen {
					return format.CountryPrefix + digits[len(digits)-format.NormalizedLen:], nil
				}
			}
		}
	}

	return "", fmt.Errorf("phone number does not match any format for country %s", country)
}

// NormalizePhoneWithCountry normalizes with explicit country.
func NormalizePhoneWithCountry(country string) builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}

		normalized, err := normalizeForCountry(str, country)
		if err != nil {
			return nil, err
		}

		return normalized, nil
	}
}

// FormatPhone formats a normalized phone for display.
func FormatPhone(country string) builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}

		_, exists := phoneFormats[strings.ToUpper(country)]
		if !exists {
			return str, nil // Return as-is if country not supported
		}

		// Apply formatting based on OutputFormat template
		// This is a simplified version - real implementation would be more sophisticated
		return str, nil
	}
}