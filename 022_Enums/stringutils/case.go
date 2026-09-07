// case.go is a SEPARATE FILE but still part of package stringutils.
// Go packages can span multiple files; anything unexported here (like
// vowels below) is visible to stringutils.go too, but not outside the package.
package stringutils

import "strings"

var vowels = "aeiouAEIOU"

// CountVowels counts vowels in s, using the package-private `vowels` var
// declared in this file — accessible from stringutils.go as well since
// they're both in package stringutils.
func CountVowels(s string) int {
	count := 0
	for _, ch := range s {
		if strings.ContainsRune(vowels, ch) {
			count++
		}
	}
	return count
}

// Title capitalizes the first letter of each word.
func Title(s string) string {
	return strings.Title(strings.ToLower(s)) //nolint:staticcheck // demo simplicity
}
