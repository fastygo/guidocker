package utils

import "strings"

// Cn joins non-empty class fragments with a single space.
func Cn(classes ...string) string {
	parts := make([]string, 0, len(classes))
	for _, className := range classes {
		trimmed := strings.TrimSpace(className)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, " ")
}
