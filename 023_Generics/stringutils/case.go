package stringutils

import "strings"

var vowels = "aeiouAEIOU"

func CountVowels(s string) int {
	count := 0
	for _, ch := range s {
		if strings.ContainsRune(vowels, ch) {
			count++
		}
	}
	return count
}

func Title(s string) string {
	return strings.Title(strings.ToLower(s))
}
