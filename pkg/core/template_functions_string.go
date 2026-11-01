// Package core provides string manipulation template functions for the todoer application.
package core

import (
	"strings"
	"text/template"
	"unicode/utf8"
)

// createStringFunctions returns a map of string manipulation template functions.
// These functions provide text transformation, searching, and formatting operations.
func createStringFunctions() template.FuncMap {
	return template.FuncMap{
		// Case transformation
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": func(s string) string {
			// Capitalise the first rune of each word. Iterating runes
			// (not slicing word[:1]) matters: a byte slice of a
			// multi-byte letter produces a broken character.
			if s == "" {
				return s
			}
			words := strings.Fields(s)
			for i, word := range words {
				lower := strings.ToLower(word)
				r, size := utf8.DecodeRuneInString(lower)
				if r == utf8.RuneError && size <= 1 {
					continue
				}
				words[i] = strings.ToUpper(string(r)) + lower[size:]
			}
			return strings.Join(words, " ")
		},

		// String operations
		"trim": strings.TrimSpace,
		"replace": func(oldValue, newValue, str string) string {
			return strings.ReplaceAll(str, oldValue, newValue)
		},
		"repeat": strings.Repeat,

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
