package transformers_test

import (
	"testing"
	"time"

	"github.com/ha1tch/queryfy/builders/transformers"
)

// --- helpers ---

func apply(t *testing.T, transformer func(interface{}) (interface{}, error), input interface{}) interface{} {
	t.Helper()
	result, err := transformer(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return result
}

func expectError(t *testing.T, transformer func(interface{}) (interface{}, error), input interface{}) {
	t.Helper()
	_, err := transformer(input)
	if err == nil {
		t.Errorf("expected error for input %v (%T), got none", input, input)
	}
}

// ======================================================================
// String transformers
// ======================================================================

func TestTrim(t *testing.T) {
	tr := transformers.Trim()
	if apply(t, tr, "  hello  ") != "hello" {
		t.Error("Trim failed")
	}
	if apply(t, tr, "\t\n  spaced  \n\t") != "spaced" {
		t.Error("Trim with tabs/newlines failed")
	}
	if apply(t, tr, "nospace") != "nospace" {
		t.Error("Trim no-op failed")
	}
	expectError(t, tr, 42) // wrong type
}

func TestLowercase(t *testing.T) {
	tr := transformers.Lowercase()
	if apply(t, tr, "HELLO") != "hello" {
		t.Error("Lowercase failed")
	}
	if apply(t, tr, "MiXeD") != "mixed" {
		t.Error("Lowercase mixed case failed")
	}
	expectError(t, tr, 42)
}

func TestUppercase(t *testing.T) {
	tr := transformers.Uppercase()
	if apply(t, tr, "hello") != "HELLO" {
		t.Error("Uppercase failed")
	}
	expectError(t, tr, 42)
}

func TestTitleCase(t *testing.T) {
	tr := transformers.TitleCase()
	result := apply(t, tr, "hello world").(string)
	if result != "Hello World" {
		t.Errorf("TitleCase: expected 'Hello World', got %q", result)
	}
	expectError(t, tr, 42)
}

func TestRemoveSpaces(t *testing.T) {
	tr := transformers.RemoveSpaces()
	if apply(t, tr, "hello world") != "helloworld" {
		t.Error("RemoveSpaces failed")
	}
	if apply(t, tr, "  a  b  c  ") != "abc" {
		t.Error("RemoveSpaces multiple failed")
	}
	expectError(t, tr, 42)
}

func TestNormalizeWhitespace(t *testing.T) {
	tr := transformers.NormalizeWhitespace()
	if apply(t, tr, "  hello   world  ") != "hello world" {
		t.Error("NormalizeWhitespace failed")
	}
	if apply(t, tr, "a\t\tb\n\nc") != "a b c" {
		t.Error("NormalizeWhitespace tabs/newlines failed")
	}
	expectError(t, tr, 42)
}

func TestReplace(t *testing.T) {
	tr := transformers.Replace("-", "_")
	if apply(t, tr, "foo-bar-baz") != "foo_bar_baz" {
		t.Error("Replace failed")
	}
	expectError(t, tr, 42)
}

func TestRemoveNonAlphanumeric(t *testing.T) {
	tr := transformers.RemoveNonAlphanumeric()
	if apply(t, tr, "hello, world! 123") != "helloworld123" {
		t.Error("RemoveNonAlphanumeric failed")
	}
	if apply(t, tr, "abc") != "abc" {
		t.Error("RemoveNonAlphanumeric no-op failed")
	}
	expectError(t, tr, 42)
}

func TestTruncate(t *testing.T) {
	tr := transformers.Truncate(5)
	if apply(t, tr, "hello world") != "hello" {
		t.Error("Truncate failed")
	}
	if apply(t, tr, "hi") != "hi" {
		t.Error("Truncate no-op for short string failed")
	}
	expectError(t, tr, 42)
}

func TestPadLeft(t *testing.T) {
	tr := transformers.PadLeft(5, '0')
	if apply(t, tr, "42") != "00042" {
		t.Error("PadLeft failed")
	}
	if apply(t, tr, "12345") != "12345" {
		t.Error("PadLeft no-op for exact length failed")
	}
	if apply(t, tr, "123456") != "123456" {
		t.Error("PadLeft no-op for longer string failed")
	}
	expectError(t, tr, 42)
}

// ======================================================================
// Number transformers
// ======================================================================

func TestToFloat64(t *testing.T) {
	tr := transformers.ToFloat64()
	if apply(t, tr, 42) != 42.0 {
		t.Error("ToFloat64 from int failed")
	}
	if apply(t, tr, int64(100)) != 100.0 {
		t.Error("ToFloat64 from int64 failed")
	}
	if apply(t, tr, "3.14") != 3.14 {
		t.Error("ToFloat64 from string failed")
	}
	if apply(t, tr, float32(1.5)).(float64) != float64(float32(1.5)) {
		t.Error("ToFloat64 from float32 failed")
	}
	expectError(t, tr, "not-a-number")
	expectError(t, tr, true)
}

func TestToInt(t *testing.T) {
	tr := transformers.ToInt()
	if apply(t, tr, 42) != 42 {
		t.Error("ToInt from int failed")
	}
	if apply(t, tr, 3.7) != 3 {
		t.Error("ToInt from float64 (truncation) failed")
	}
	if apply(t, tr, "100") != 100 {
		t.Error("ToInt from string failed")
	}
	if apply(t, tr, "3.14") != 3 {
		t.Error("ToInt from float string failed")
	}
	expectError(t, tr, "not-a-number")
	expectError(t, tr, true)
}

func TestRound(t *testing.T) {
	tr := transformers.Round(2)
	result := apply(t, tr, 3.14159).(float64)
	if result != 3.14 {
		t.Errorf("Round(2): expected 3.14, got %v", result)
	}
	result = apply(t, tr, 2.005).(float64)
	// Note: floating point — 2.005 rounds to 2.01 with math.Round
	if result != 2.0 && result != 2.01 {
		t.Errorf("Round(2) edge: expected ~2.00 or 2.01, got %v", result)
	}
}

func TestClamp(t *testing.T) {
	tr := transformers.Clamp(0, 100)
	if apply(t, tr, 50.0) != 50.0 {
		t.Error("Clamp in range failed")
	}
	if apply(t, tr, -10.0) != 0.0 {
		t.Error("Clamp below min failed")
	}
	if apply(t, tr, 200.0) != 100.0 {
		t.Error("Clamp above max failed")
	}
	if apply(t, tr, 0.0) != 0.0 {
		t.Error("Clamp at min boundary failed")
	}
	if apply(t, tr, 100.0) != 100.0 {
		t.Error("Clamp at max boundary failed")
	}
}

func TestPercentage(t *testing.T) {
	tr := transformers.Percentage()
	if apply(t, tr, 0.15) != 15.0 {
		t.Error("Percentage failed")
	}
	if apply(t, tr, 1.0) != 100.0 {
		t.Error("Percentage 100% failed")
	}
}

func TestFromPercentage(t *testing.T) {
	tr := transformers.FromPercentage()
	if apply(t, tr, 15.0) != 0.15 {
		t.Error("FromPercentage failed")
	}
	if apply(t, tr, 100.0) != 1.0 {
		t.Error("FromPercentage 100% failed")
	}
}

// ======================================================================
// Common transformers
// ======================================================================

func TestToString(t *testing.T) {
	tr := transformers.ToString()
	if apply(t, tr, "hello") != "hello" {
		t.Error("ToString from string failed")
	}
	if apply(t, tr, 42) != "42" {
		t.Error("ToString from int failed")
	}
	if apply(t, tr, true) != "true" {
		t.Error("ToString from bool failed")
	}
	if apply(t, tr, 3.14) != "3.14" {
		t.Error("ToString from float64 failed")
	}
	if apply(t, tr, int64(99)) != "99" {
		t.Error("ToString from int64 failed")
	}
}

func TestToBoolean(t *testing.T) {
	tr := transformers.ToBoolean()
	if apply(t, tr, true) != true {
		t.Error("ToBoolean from true failed")
	}
	if apply(t, tr, false) != false {
		t.Error("ToBoolean from false failed")
	}
	if apply(t, tr, "true") != true {
		t.Error("ToBoolean from 'true' failed")
	}
	if apply(t, tr, "false") != false {
		t.Error("ToBoolean from 'false' failed")
	}
	if apply(t, tr, "yes") != true {
		t.Error("ToBoolean from 'yes' failed")
	}
	if apply(t, tr, "no") != false {
		t.Error("ToBoolean from 'no' failed")
	}
	if apply(t, tr, "1") != true {
		t.Error("ToBoolean from '1' failed")
	}
	if apply(t, tr, "0") != false {
		t.Error("ToBoolean from '0' failed")
	}
	if apply(t, tr, 1) != true {
		t.Error("ToBoolean from int 1 failed")
	}
	if apply(t, tr, 0) != false {
		t.Error("ToBoolean from int 0 failed")
	}
	expectError(t, tr, "maybe")
	expectError(t, tr, []int{1})
}

func TestDefault(t *testing.T) {
	tr := transformers.Default("fallback")
	if apply(t, tr, nil) != "fallback" {
		t.Error("Default for nil failed")
	}
	if apply(t, tr, "") != "fallback" {
		t.Error("Default for empty string failed")
	}
	if apply(t, tr, "actual") != "actual" {
		t.Error("Default with real value should not replace")
	}
	if apply(t, tr, 42) != 42 {
		t.Error("Default with non-nil non-string should not replace")
	}
}

func TestRemoveCurrencySymbols(t *testing.T) {
	tr := transformers.RemoveCurrencySymbols()
	if apply(t, tr, "$1,234.56") != "1234.56" {
		t.Error("RemoveCurrencySymbols dollar failed")
	}
	if apply(t, tr, "€99.99") != "99.99" {
		t.Error("RemoveCurrencySymbols euro failed")
	}
	if apply(t, tr, "£1,000") != "1000" {
		t.Error("RemoveCurrencySymbols pound failed")
	}
	if apply(t, tr, "42.00") != "42.00" {
		t.Error("RemoveCurrencySymbols no-op failed")
	}
	expectError(t, tr, 42)
}

func TestChain(t *testing.T) {
	tr := transformers.Chain(
		transformers.Trim(),
		transformers.Lowercase(),
		transformers.RemoveSpaces(),
	)
	if apply(t, tr, "  Hello World  ") != "helloworld" {
		t.Error("Chain failed")
	}
}

func TestChain_ErrorPropagation(t *testing.T) {
	tr := transformers.Chain(
		transformers.Trim(),
		transformers.ToInt(), // will fail on non-numeric string
	)
	expectError(t, tr, "  not-a-number  ")
}

func TestConditional(t *testing.T) {
	isLong := func(v interface{}) bool {
		s, ok := v.(string)
		return ok && len(s) > 10
	}
	tr := transformers.Conditional(isLong, transformers.Truncate(10))

	// Long string gets truncated
	if apply(t, tr, "this is a very long string") != "this is a " {
		t.Error("Conditional: should truncate long string")
	}
	// Short string passes through
	if apply(t, tr, "short") != "short" {
		t.Error("Conditional: should not modify short string")
	}
}

// ======================================================================
// Date transformers
// ======================================================================

func TestParseDate(t *testing.T) {
	tr := transformers.ParseDate("2006-01-02")
	result := apply(t, tr, "2024-06-15")
	tm, ok := result.(time.Time)
	if !ok {
		t.Fatalf("ParseDate: expected time.Time, got %T", result)
	}
	if tm.Year() != 2024 || tm.Month() != 6 || tm.Day() != 15 {
		t.Errorf("ParseDate: wrong date: %v", tm)
	}

	expectError(t, tr, "not-a-date")
	expectError(t, tr, "15/06/2024") // wrong format
	expectError(t, tr, 42)
}

func TestToISO8601_FromTime(t *testing.T) {
	tr := transformers.ToISO8601()
	input := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	result := apply(t, tr, input).(string)
	if result != "2024-06-15T10:30:00Z" {
		t.Errorf("ToISO8601 from time: expected '2024-06-15T10:30:00Z', got %q", result)
	}
}

func TestToISO8601_FromString(t *testing.T) {
	tr := transformers.ToISO8601()
	result := apply(t, tr, "2024-06-15").(string)
	// Should parse date-only and produce RFC3339
	if result != "2024-06-15T00:00:00Z" {
		t.Errorf("ToISO8601 from string: expected '2024-06-15T00:00:00Z', got %q", result)
	}

	expectError(t, tr, "garbage")
	expectError(t, tr, 42)
}

func TestDateFormat(t *testing.T) {
	tr := transformers.DateFormat("2006-01-02", "01/02/2006")
	result := apply(t, tr, "2024-06-15").(string)
	if result != "06/15/2024" {
		t.Errorf("DateFormat: expected '06/15/2024', got %q", result)
	}

	expectError(t, tr, "15/06/2024") // wrong input format
	expectError(t, tr, 42)
}

func TestToTimezone_FromTime(t *testing.T) {
	tr := transformers.ToTimezone("America/New_York")
	utcTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	result := apply(t, tr, utcTime).(time.Time)
	if result.Location().String() != "America/New_York" {
		t.Errorf("ToTimezone: expected America/New_York, got %s", result.Location())
	}
}

func TestToTimezone_FromString(t *testing.T) {
	tr := transformers.ToTimezone("UTC")
	result := apply(t, tr, "2024-06-15T12:00:00+05:00").(time.Time)
	if result.Location().String() != "UTC" {
		t.Errorf("ToTimezone from string: expected UTC, got %s", result.Location())
	}
	if result.Hour() != 7 {
		t.Errorf("ToTimezone: expected hour 7 in UTC, got %d", result.Hour())
	}
}

func TestToTimezone_InvalidTimezone(t *testing.T) {
	tr := transformers.ToTimezone("Not/A/Timezone")
	expectError(t, tr, time.Now())
}

func TestStartOfDay(t *testing.T) {
	tr := transformers.StartOfDay()

	// From time.Time
	input := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)
	result := apply(t, tr, input).(time.Time)
	if result.Hour() != 0 || result.Minute() != 0 || result.Second() != 0 {
		t.Errorf("StartOfDay from time: expected 00:00:00, got %v", result)
	}
	if result.Day() != 15 {
		t.Errorf("StartOfDay: day should be preserved, got %d", result.Day())
	}

	// From string
	result2 := apply(t, tr, "2024-06-15").(time.Time)
	if result2.Hour() != 0 || result2.Day() != 15 {
		t.Errorf("StartOfDay from string: unexpected result %v", result2)
	}

	expectError(t, tr, 42)
}

func TestEndOfDay(t *testing.T) {
	tr := transformers.EndOfDay()

	input := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)
	result := apply(t, tr, input).(time.Time)
	if result.Hour() != 23 || result.Minute() != 59 || result.Second() != 59 {
		t.Errorf("EndOfDay: expected 23:59:59, got %02d:%02d:%02d",
			result.Hour(), result.Minute(), result.Second())
	}
	if result.Day() != 15 {
		t.Errorf("EndOfDay: day should be preserved, got %d", result.Day())
	}

	// From string
	result2 := apply(t, tr, "2024-06-15").(time.Time)
	if result2.Hour() != 23 || result2.Day() != 15 {
		t.Errorf("EndOfDay from string: unexpected result %v", result2)
	}

	expectError(t, tr, 42)
}

// ======================================================================
// Phone transformers
// ======================================================================

func TestNormalizePhone_US(t *testing.T) {
	tr := transformers.NormalizePhone("US")

	// Various US formats
	cases := []struct {
		input    string
		contains string // partial match since exact format may vary
	}{
		{"(555) 123-4567", "5551234567"},
		{"555-123-4567", "5551234567"},
		{"5551234567", "5551234567"},
		{"+1 555 123 4567", "5551234567"},
		{"1 (555) 123-4567", "5551234567"},
	}

	for _, tc := range cases {
		result, err := tr(tc.input)
		if err != nil {
			t.Errorf("NormalizePhone(%q): unexpected error: %v", tc.input, err)
			continue
		}
		str, ok := result.(string)
		if !ok {
			t.Errorf("NormalizePhone(%q): expected string, got %T", tc.input, result)
			continue
		}
		// Strip everything but digits for comparison
		digits := ""
		for _, c := range str {
			if c >= '0' && c <= '9' {
				digits += string(c)
			}
		}
		if len(digits) < 10 {
			t.Errorf("NormalizePhone(%q): result %q has fewer than 10 digits", tc.input, str)
		}
	}

	expectError(t, tr, 42) // wrong type
}

func TestNormalizePhone_UK(t *testing.T) {
	tr := transformers.NormalizePhone("UK")
	// UK mobile format — matches the 07NNNNNNNNN pattern
	result, err := tr("07911123456")
	if err != nil {
		t.Fatalf("NormalizePhone UK: unexpected error: %v", err)
	}
	str := result.(string)
	if len(str) == 0 {
		t.Error("NormalizePhone UK: empty result")
	}
}

func TestNormalizePhoneWithCountry(t *testing.T) {
	tr := transformers.NormalizePhoneWithCountry("US")
	result, err := tr("555-123-4567")
	if err != nil {
		t.Fatalf("NormalizePhoneWithCountry: unexpected error: %v", err)
	}
	str := result.(string)
	if len(str) == 0 {
		t.Error("NormalizePhoneWithCountry: empty result")
	}
	expectError(t, tr, 42)
}

func TestFormatPhone_US(t *testing.T) {
	tr := transformers.FormatPhone("US")
	result, err := tr("5551234567")
	if err != nil {
		t.Fatalf("FormatPhone US: unexpected error: %v", err)
	}
	str := result.(string)
	if len(str) == 0 {
		t.Error("FormatPhone US: empty result")
	}
	expectError(t, tr, 42)
}
