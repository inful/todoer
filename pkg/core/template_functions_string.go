// Package core provides string manipulation template functions for the todoer application.
package core

import (
	"strings"
	"text/template"
)

// createStringFunctions returns a map of string manipulation template functions.
// These functions provide text transformation, searching, and formatting operations.
func createStringFunctions() template.FuncMap {
	return template.FuncMap{
		// Case transformation
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": func(s string) string {
			// Simple title case implementation - capitalize first letter of each word
			if s == "" {
				return s
			}
			words := strings.Fields(s)
			for i, word := range words {
				if len(word) > 0 {
					words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
				}
			}
			return strings.Join(words, " ")
		},

		// String operations
		"trim": strings.TrimSpace,
		"replace": func(old, new, str string) string {
			return strings.ReplaceAll(str, old, new)
		},
		"repeat": strings.Repeat,
		"len": func(s string) int {
			return len(s)
		},

		// String searching
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,

		// String splitting and joining
		"split": func(sep, str string) []string {
			return strings.Split(str, sep)
		},
		"join": func(sep string, strs []string) string {
			return strings.Join(strs, sep)
		},
	}
}
