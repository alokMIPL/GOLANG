package stringutils_test

import (
	"testing"

	"go-packages-demo/stringutils"
)

func TestReverse(t *testing.T) {
	got := stringutils.Reverse("golang")
	want := "gnalog"
	if got != want {
		t.Errorf("Reverse() = %q; want %q", got, want)
	}
}

func TestIsPalindrome(t *testing.T) {
	cases := map[string]bool{
		"racecar":                     true,
		"A man a plan a canal Panama": true,
		"golang":                      false,
	}
	for input, want := range cases {
		if got := stringutils.IsPalindrome(input); got != want {
			t.Errorf("IsPalindrome(%q) = %v; want %v", input, got, want)
		}
	}
}

func TestCountVowels(t *testing.T) {
	if got := stringutils.CountVowels("Hello World"); got != 3 {
		t.Errorf("CountVowels() = %d; want 3", got)
	}
}
