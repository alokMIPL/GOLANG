package greet

import "strings"

// If we use Capital Letter then it is Exported function.
func Hello(name string) string {
	clean := normalizeName(name)

	return "Hello," + clean
}

func normalizeName(name string) string {
	n := strings.TrimSpace(name)

	if n == "" {
		return "Guest"
	}
	return strings.ToUpper(n)
}
