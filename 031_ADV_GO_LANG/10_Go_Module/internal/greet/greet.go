package greet

import "strings"

func normalizeName(name string) string {
	n := strings.TrimSpace(name)

	if n == "" {
		return "Guest"
	}
	return strings.ToUpper(n)
}
