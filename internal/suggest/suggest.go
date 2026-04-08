// Package suggest provides Levenshtein-based suggestions for unknown commands and flags.
package suggest

import "strings"

// EditDistance calculates the Levenshtein distance between two strings.
func EditDistance(a, b string) int {
	a = strings.ToLower(a)
	b = strings.ToLower(b)

	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create matrix
	rows := len(a) + 1
	cols := len(b) + 1
	prev := make([]int, cols)
	curr := make([]int, cols)

	for j := 0; j < cols; j++ {
		prev[j] = j
	}

	for i := 1; i < rows; i++ {
		curr[0] = i
		for j := 1; j < cols; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			insert := curr[j-1] + 1
			del := prev[j] + 1
			sub := prev[j-1] + cost
			curr[j] = min(insert, min(del, sub))
		}
		prev, curr = curr, prev
	}

	return prev[cols-1]
}

// FindClosest returns the closest match from candidates within maxDistance.
// Returns empty string if no match is close enough.
func FindClosest(input string, candidates []string, maxDistance int) string {
	matches := FindClosestN(input, candidates, maxDistance, 1)
	if len(matches) == 0 {
		return ""
	}
	return matches[0]
}

// FindClosestN returns up to n closest matches from candidates within maxDistance,
// sorted by distance (closest first).
func FindClosestN(input string, candidates []string, maxDistance, n int) []string {
	type scored struct {
		name string
		dist int
	}

	var matches []scored
	for _, c := range candidates {
		d := EditDistance(input, c)
		if d <= maxDistance && d > 0 {
			matches = append(matches, scored{name: c, dist: d})
		}
	}

	// Sort by distance (insertion sort — small N)
	for i := 1; i < len(matches); i++ {
		key := matches[i]
		j := i - 1
		for j >= 0 && matches[j].dist > key.dist {
			matches[j+1] = matches[j]
			j--
		}
		matches[j+1] = key
	}

	if len(matches) > n {
		matches = matches[:n]
	}

	result := make([]string, len(matches))
	for i, m := range matches {
		result[i] = m.name
	}
	return result
}

// FormatSuggestion returns a user-friendly suggestion message.
// Returns empty string if no close match is found.
func FormatSuggestion(unknown string, candidates []string, maxDistance int) string {
	closest := FindClosest(unknown, candidates, maxDistance)
	if closest == "" {
		return ""
	}
	return "Did you mean '" + closest + "'?"
}

// FormatSuggestions returns a message with multiple suggestions.
// Returns empty string if no matches found.
func FormatSuggestions(unknown string, candidates []string, maxDistance, maxSuggestions int) string {
	matches := FindClosestN(unknown, candidates, maxDistance, maxSuggestions)
	if len(matches) == 0 {
		return ""
	}
	if len(matches) == 1 {
		return "Did you mean '" + matches[0] + "'?"
	}
	parts := make([]string, len(matches))
	for i, m := range matches {
		parts[i] = "'" + m + "'"
	}
	return "Did you mean one of: " + strings.Join(parts, ", ") + "?"
}
