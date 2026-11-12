package utils

import (
	"regexp"
	"strings"
)

// phoneRunRe matches runs of 10 or more digits (optionally with a leading '+'),
// used to scrub phone-like numbers out of free-form log blobs.
var phoneRunRe = regexp.MustCompile(`\+?\d{10,}`)

// pinRe matches a standalone 6-digit Indian PIN code.
var pinRe = regexp.MustCompile(`\b\d{6}\b`)

// MaskPhone masks a phone number, keeping only the last 2 digits.
// e.g. "+919876543210" -> "+91*********10". Empty input returns "".
func MaskPhone(phone string) string {
	p := strings.TrimSpace(phone)
	if p == "" {
		return ""
	}
	total := 0
	for _, r := range p {
		if r >= '0' && r <= '9' {
			total++
		}
	}
	const keep = 2
	if total <= keep {
		return strings.Repeat("*", total)
	}
	cutoff := total - keep
	var b strings.Builder
	seen := 0
	for _, r := range p {
		if r >= '0' && r <= '9' {
			if seen < cutoff {
				b.WriteByte('*')
			} else {
				b.WriteRune(r)
			}
			seen++
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// MaskName keeps the first character of each name token and masks the rest.
// e.g. "John Doe" -> "J*** D**". Empty input returns "".
func MaskName(name string) string {
	n := strings.TrimSpace(name)
	if n == "" {
		return ""
	}
	tokens := strings.Fields(n)
	for i, t := range tokens {
		r := []rune(t)
		if len(r) <= 1 {
			tokens[i] = string(r)
			continue
		}
		tokens[i] = string(r[0]) + strings.Repeat("*", len(r)-1)
	}
	return strings.Join(tokens, " ")
}

// MaskAddress redacts a free-form address, keeping only a trailing 6-digit PIN
// code (if present) for coarse debugging. Everything else is removed.
func MaskAddress(addr string) string {
	a := strings.TrimSpace(addr)
	if a == "" {
		return ""
	}
	if pin := pinRe.FindString(a); pin != "" {
		return "[redacted address] PIN " + pin
	}
	return "[redacted address]"
}

// MaskPhonesInText scrubs phone-like digit runs (10+ digits) from a free-form
// blob such as an API request/response body, keeping only the last 2 digits of
// each run. Best-effort: it does not guarantee that no other PII remains.
func MaskPhonesInText(s string) string {
	return phoneRunRe.ReplaceAllStringFunc(s, func(m string) string {
		plus := ""
		digits := m
		if strings.HasPrefix(digits, "+") {
			plus = "+"
			digits = digits[1:]
		}
		if len(digits) <= 2 {
			return plus + strings.Repeat("*", len(digits))
		}
		return plus + strings.Repeat("*", len(digits)-2) + digits[len(digits)-2:]
	})
}
