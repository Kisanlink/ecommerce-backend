package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidatePhoneNumber validates and normalizes an Indian mobile number
func ValidatePhoneNumber(phone string) (string, error) {
	if phone == "" {
		return "", fmt.Errorf("phone number cannot be empty")
	}

	// Extract only digits
	digits := extractDigits(phone)

	// Handle international format (+91 prefix)
	if strings.HasPrefix(digits, "91") && len(digits) == 12 {
		digits = digits[2:] // Remove country code
	}

	// Must be exactly 10 digits for Indian mobile numbers
	if len(digits) != 10 {
		return "", fmt.Errorf("phone number must be exactly 10 digits, got %d digits", len(digits))
	}

	// Validate Indian mobile number format - must start with 6, 7, 8, or 9
	firstDigit := digits[0]
	if firstDigit < '6' || firstDigit > '9' {
		return "", fmt.Errorf("Indian mobile numbers must start with 6, 7, 8, or 9, got %c", firstDigit)
	}

	// Additional validation: second digit restrictions
	if firstDigit == '6' {
		// 6 series: 60-69, but 60 and 61 are not common
		secondDigit := digits[1]
		if secondDigit < '0' || secondDigit > '9' {
			return "", fmt.Errorf("invalid mobile number format")
		}
	}

	return digits, nil
}

// extractDigits extracts only digits from a string
func extractDigits(s string) string {
	re := regexp.MustCompile(`\d`)
	matches := re.FindAllString(s, -1)
	return strings.Join(matches, "")
}
