// Package core provides utility template functions for the todoer application.
package core

import (
	"math/rand"
	"strings"
	"text/template"
)

// isEmptyValue reports whether val is the zero/empty value for its type.
func isEmptyValue(val any) bool {
	if val == nil {
		return true
	}
	switch v := val.(type) {
	case string:
		return v == ""
	case []string:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	case int:
		return v == 0
	default:
		return false
	}
}

// shuffleStrings shuffles a slice of strings in place using the global RNG.
func shuffleStrings(s []string) {
	rand.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
}

// createUtilityFunctions returns a map of utility template functions.
// These functions provide conditional logic, collections, arithmetic, and shuffling operations.
func createUtilityFunctions() template.FuncMap {
	return template.FuncMap{
		// Conditional and default values
		"default": func(defaultVal any, val any) any {
			if val == nil || val == "" {
				return defaultVal
			}
			return val
		},
		"empty": func(val any) bool {
			return isEmptyValue(val)
		},
		"notEmpty": func(val any) bool {
			return !isEmptyValue(val)
		},

		// Collection functions
		"seq": func(start, end int) []int {
			if start > end {
				return []int{}
			}
			result := make([]int, end-start+1)
			for i := range result {
				result[i] = start + i
			}
			return result
		},
		"dict": func(values ...any) map[string]any {
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]any)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil
				}
				dict[key] = values[i+1]
			}
			return dict
		},

		// Shuffling functions
		"shuffle": func(text string) string {
			// Split the text into lines, filter out empty lines
			lines := strings.Split(strings.TrimSpace(text), "\n")
			var nonEmptyLines []string
			for _, line := range lines {
				if trimmed := strings.TrimSpace(line); trimmed != "" {
					nonEmptyLines = append(nonEmptyLines, line)
				}
			}

			if len(nonEmptyLines) <= 1 {
				return text
			}

			shuffled := make([]string, len(nonEmptyLines))
			copy(shuffled, nonEmptyLines)
			shuffleStrings(shuffled)

			return strings.Join(shuffled, "\n")
		},
		"shuffleLines": func(lines []string) []string {
			if len(lines) <= 1 {
				return lines
			}

			shuffled := make([]string, len(lines))
			copy(shuffled, lines)
			shuffleStrings(shuffled)

			return shuffled
		},

		// Arithmetic functions
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0 // Prevent division by zero
			}
			return a / b
		},
	}
}
