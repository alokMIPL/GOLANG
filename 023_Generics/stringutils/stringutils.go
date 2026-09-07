// Package stringutils provides small string helper functions.
package stringutils

import "strings"

// Reverse returns the input string reversed.
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// IsPalindrome checks if s reads the same forwards and backwards
// (case-insensitive, ignoring spaces).
func IsPalindrome(s string) bool {
	clean := strings.ToLower(strings.ReplaceAll(s, " ", ""))
	return clean == Reverse(clean)
}
