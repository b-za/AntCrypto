package utils

import "strings"

// HasPrefixCaseInsensitive checks if string s has prefix, ignoring case
func HasPrefixCaseInsensitive(s, prefix string) bool {
	return strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix))
}

// EqualCaseInsensitive checks if two strings are equal, ignoring case
func EqualCaseInsensitive(s1, s2 string) bool {
	return strings.ToLower(s1) == strings.ToLower(s2)
}

// ContainsCaseInsensitive checks if string contains substring, ignoring case
func ContainsCaseInsensitive(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// TrimSpaceAndCase trims whitespace AND normalizes case
// Useful for standardized comparisons while preserving original
func TrimSpaceAndCase(s string) (original, normalized string) {
	original = strings.TrimSpace(s)
	normalized = strings.ToLower(original)
	return
}
