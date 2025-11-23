// Package core provides utility template functions for the todoer application.
package core

import (
	"math/rand"
	"strings"
	"text/template"
	"time"
)

// createUtilityFunctions returns a map of utility template functions.
// These functions provide conditional logic, collections, arithmetic, and shuffling operations.
func createUtilityFunctions() template.FuncMap {
	return template.FuncMap{
		// Conditional and default values
		"default": func(defaultVal interface{}, val interface{}) interface{} {
			if val == nil || val == "" {
				return defaultVal
			}
			return val
		},
		"empty": func(val interface{}) bool {
			if val == nil {
				return true
			}
			switch v := val.(type) {
			case string:
				return v == ""
			case []string:
				return len(v) == 0
			case map[string]interface{}:
				return len(v) == 0
			case int:
				return v == 0
			default:
				return false
			}
		},
		"notEmpty": func(val interface{}) bool {
			if val == nil {
				return false
			}
			switch v := val.(type) {
			case string:
				return v != ""
			case []string:
				return len(v) > 0
			case map[string]interface{}:
				return len(v) > 0
			case int:
				return v != 0
			default:
				return true
			}
		},

		// Collection functions
		"seq": func(start, end int) []int {
			if start > end {
				return []int{}
			}
			result := make([]int, end-start+1)
			for i := 0; i < len(result); i++ {
				result[i] = start + i
			}
			return result
		},
		"dict": func(values ...interface{}) map[string]interface{} {
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]interface{})
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

			// If we have no lines or only one line, return as-is
			if len(nonEmptyLines) <= 1 {
				return text
			}

			// Create a copy for shuffling
			shuffled := make([]string, len(nonEmptyLines))
			copy(shuffled, nonEmptyLines)

			// Shuffle using Fisher-Yates algorithm
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for i := len(shuffled) - 1; i > 0; i-- {
				j := r.Intn(i + 1)
				shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
			}

			return strings.Join(shuffled, "\n")
		},
		"shuffleLines": func(lines []string) []string {
			// Create a copy for shuffling
			if len(lines) <= 1 {
				return lines
			}

			shuffled := make([]string, len(lines))
			copy(shuffled, lines)

			// Shuffle using Fisher-Yates algorithm
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for i := len(shuffled) - 1; i > 0; i-- {
				j := r.Intn(i + 1)
				shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
			}

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
