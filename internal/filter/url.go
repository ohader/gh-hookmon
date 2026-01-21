package filter

import "strings"

// MatchesPattern checks if a URL matches the given pattern (case-insensitive substring matching)
func MatchesPattern(url, pattern string) bool {
	if pattern == "" {
		return true // No filter means match all
	}
	return strings.Contains(
		strings.ToLower(url),
		strings.ToLower(pattern),
	)
}
