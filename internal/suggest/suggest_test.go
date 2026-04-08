package suggest

import (
	"testing"
)

func TestEditDistance(t *testing.T) {
	tests := []struct {
		name     string
		a, b     string
		expected int
	}{
		{"identical strings", "hello", "hello", 0},
		{"empty strings", "", "", 0},
		{"one empty", "hello", "", 5},
		{"other empty", "", "world", 5},
		{"single substitution", "cat", "bat", 1},
		{"single insertion", "cat", "cats", 1},
		{"single deletion", "cats", "cat", 1},
		{"transposition", "ab", "ba", 2}, // Levenshtein, not Damerau
		{"completely different", "abc", "xyz", 3},
		{"case insensitive", "Hello", "hello", 0},
		{"groups typo", "grops", "groups", 1},
		{"policies typo", "polcies", "policies", 1},
		{"boundaries typo", "boundries", "boundaries", 1},
		{"longer strings", "environment", "enviroment", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EditDistance(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("EditDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestEditDistance_Symmetry(t *testing.T) {
	pairs := [][2]string{
		{"hello", "world"},
		{"groups", "grops"},
		{"cat", ""},
		{"abc", "xyz"},
	}

	for _, pair := range pairs {
		d1 := EditDistance(pair[0], pair[1])
		d2 := EditDistance(pair[1], pair[0])
		if d1 != d2 {
			t.Errorf("EditDistance is not symmetric: (%q,%q)=%d but (%q,%q)=%d",
				pair[0], pair[1], d1, pair[1], pair[0], d2)
		}
	}
}

func TestFindClosest(t *testing.T) {
	resources := []string{
		"groups", "users", "policies", "bindings",
		"environments", "boundaries", "service-users",
	}

	tests := []struct {
		name        string
		input       string
		maxDistance  int
		expected    string
	}{
		{"exact match excluded", "groups", 3, ""},
		{"single typo", "grops", 3, "groups"},
		{"two typos", "polcies", 3, "policies"},
		{"close to boundaries", "boundries", 3, "boundaries"},
		{"close to users", "usrs", 3, "users"},
		{"too far away", "zzzzz", 3, ""},
		{"empty input", "", 3, ""},
		{"close to environments", "enviroments", 3, "environments"},
		{"service users typo", "service-user", 3, "service-users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindClosest(tt.input, resources, tt.maxDistance)
			if got != tt.expected {
				t.Errorf("FindClosest(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFindClosestN(t *testing.T) {
	candidates := []string{"get", "set", "let", "bet", "pet", "delete"}

	tests := []struct {
		name       string
		input      string
		maxDist    int
		n          int
		wantLen    int
		wantFirst  string
	}{
		{"multiple close matches", "met", 2, 3, 3, "get"},
		{"limit to 1", "met", 2, 1, 1, "get"},
		{"no matches", "zzz", 1, 3, 0, ""},
		{"close matches", "delet", 2, 5, 2, "delete"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindClosestN(tt.input, candidates, tt.maxDist, tt.n)
			if len(got) != tt.wantLen {
				t.Errorf("FindClosestN(%q) returned %d matches, want %d: %v",
					tt.input, len(got), tt.wantLen, got)
			}
			if tt.wantFirst != "" && len(got) > 0 && got[0] != tt.wantFirst {
				t.Errorf("FindClosestN(%q)[0] = %q, want %q", tt.input, got[0], tt.wantFirst)
			}
		})
	}
}

func TestFindClosestN_SortedByDistance(t *testing.T) {
	candidates := []string{"abc", "abd", "xyz", "abcde"}
	got := FindClosestN("abc", candidates, 3, 10)

	// Verify results are sorted by distance
	for i := 1; i < len(got); i++ {
		d1 := EditDistance("abc", got[i-1])
		d2 := EditDistance("abc", got[i])
		if d1 > d2 {
			t.Errorf("results not sorted: %q (dist=%d) before %q (dist=%d)",
				got[i-1], d1, got[i], d2)
		}
	}
}

func TestFormatSuggestion(t *testing.T) {
	candidates := []string{"groups", "users", "policies"}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"has suggestion", "grops", "Did you mean 'groups'?"},
		{"no suggestion", "zzzzz", ""},
		{"close to users", "usrs", "Did you mean 'users'?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSuggestion(tt.input, candidates, 3)
			if got != tt.expected {
				t.Errorf("FormatSuggestion(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFormatSuggestions(t *testing.T) {
	candidates := []string{"get", "set", "let", "bet"}

	tests := []struct {
		name     string
		input    string
		maxN     int
		contains string
	}{
		{"single suggestion", "delet", 3, "Did you mean 'let'?"},
		{"multiple suggestions", "met", 3, "Did you mean one of:"},
		{"no suggestions", "zzz", 3, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSuggestions(tt.input, candidates, 2, tt.maxN)
			if tt.contains == "" && got != "" {
				t.Errorf("FormatSuggestions(%q) = %q, want empty", tt.input, got)
			}
			if tt.contains != "" && len(got) == 0 {
				t.Errorf("FormatSuggestions(%q) = empty, want containing %q", tt.input, tt.contains)
			}
		})
	}
}

func TestFindClosest_EmptyCandidates(t *testing.T) {
	got := FindClosest("anything", nil, 3)
	if got != "" {
		t.Errorf("FindClosest with nil candidates = %q, want empty", got)
	}

	got = FindClosest("anything", []string{}, 3)
	if got != "" {
		t.Errorf("FindClosest with empty candidates = %q, want empty", got)
	}
}

func TestEditDistance_Unicode(t *testing.T) {
	// Ensure it handles multi-byte characters without panic
	d := EditDistance("hello", "hellö")
	if d < 1 {
		t.Errorf("EditDistance with unicode should be >= 1, got %d", d)
	}
}
