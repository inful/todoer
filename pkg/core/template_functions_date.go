// Package core provides date-related template functions for the todoer application.
package core

import (
	"text/template"
	"time"
)

// weekdayChecker returns a template function that checks whether a date string
// falls on the given weekday.
func weekdayChecker(wd time.Weekday) func(string) bool {
	return func(dateStr string) bool {
		date, err := time.Parse(DateFormat, dateStr)
		if err != nil {
			return false
		}
		return date.Weekday() == wd
	}
}

// createDateFunctions returns a map of date-related template functions.
// These functions provide date arithmetic, formatting, and weekday operations.
func createDateFunctions() template.FuncMap {
	addDays := func(dateStr string, days int) string {
		date, err := time.Parse(DateFormat, dateStr)
		if err != nil {
			return dateStr // Return original on error
		}
		return date.AddDate(0, 0, days).Format(DateFormat)
	}

	return template.FuncMap{
		// Date arithmetic functions
		"addDays": addDays,
		"subDays": func(dateStr string, days int) string {
			return addDays(dateStr, -days)
		},
		"addWeeks": func(dateStr string, weeks int) string {
			date, err := time.Parse(DateFormat, dateStr)
			if err != nil {
				return dateStr // Return original on error
			}
			return date.AddDate(0, 0, weeks*7).Format(DateFormat)
		},
		"addMonths": func(dateStr string, months int) string {
			date, err := time.Parse(DateFormat, dateStr)
			if err != nil {
				return dateStr // Return original on error
			}
			return date.AddDate(0, months, 0).Format(DateFormat)
		},
		"formatDate": func(dateStr, format string) string {
			date, err := time.Parse(DateFormat, dateStr)
			if err != nil {
				return dateStr // Return original on error
			}
			return date.Format(format)
		},
		"weekday": func(dateStr string) string {
			date, err := time.Parse(DateFormat, dateStr)
			if err != nil {
				return "" // Return empty on error
			}
			return date.Weekday().String()
		},
		"isWeekend": func(dateStr string) bool {
			date, err := time.Parse(DateFormat, dateStr)
			if err != nil {
				return false // Return false on error
			}
			weekday := date.Weekday()
			return weekday == time.Saturday || weekday == time.Sunday
		},
		"daysDiff": func(dateStr1, dateStr2 string) int {
			date1, err1 := time.Parse(DateFormat, dateStr1)
			date2, err2 := time.Parse(DateFormat, dateStr2)
			if err1 != nil || err2 != nil {
				return 0 // Return 0 on error
			}
			return int(date2.Sub(date1).Hours() / 24)
		},

		// Day of week checking functions
		"isMonday":    weekdayChecker(time.Monday),
		"isTuesday":   weekdayChecker(time.Tuesday),
		"isWednesday": weekdayChecker(time.Wednesday),
		"isThursday":  weekdayChecker(time.Thursday),
		"isFriday":    weekdayChecker(time.Friday),
		"isSaturday":  weekdayChecker(time.Saturday),
		"isSunday":    weekdayChecker(time.Sunday),
	}
}
