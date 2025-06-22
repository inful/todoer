// Package core provides shared file processing functionality for the todoer application.
package core

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// Constants for file processing
const (
	// BlankLineSeparator is the sequence that separates sections
	BlankLineSeparator = "\n\n"
	// MovedToTemplate is the template for moved todos message
	MovedToTemplate = "Moved to [[%s]]"
)

// Pre-compiled regex for better performance
var (
	excessiveBlankLinesRegex = regexp.MustCompile(`\n{3,}`)
)

// ExtractDateFromFrontmatter extracts the date from the frontmatter using a configurable key.
// If no date is found, it returns today's date as a fallback.
func ExtractDateFromFrontmatter(content string, dateKey string) (string, error) {
	if content == "" {
		return time.Now().Format(DateFormat), nil
	}

	// Use dynamic regex for the configured key
	regex := BuildFrontmatterDateRegex(dateKey)
	matches := regex.FindStringSubmatch(content)

	if len(matches) < 2 {
		// If no date found in frontmatter, use today's date
		return time.Now().Format(DateFormat), nil
	}

	// Validate the extracted date
	extractedDate := matches[1]
	if err := ValidateDate(extractedDate); err != nil {
		return "", fmt.Errorf("invalid date in frontmatter: %w", err)
	}

	return extractedDate, nil
}

// ExtractTodosSection extracts the Todos section from the file content.
// It returns three parts: content before Todos, the Todos section content, and content after Todos.
// The function expects a specific format with a blank line after the Todos header.
func ExtractTodosSection(content string) (string, string, string, error) {
	if content == "" {
		return "", "", "", fmt.Errorf("content cannot be empty")
	}

	// Find the Todos section header
	todosHeaderIndex := strings.Index(content, TodosHeader)
	if todosHeaderIndex == -1 {
		return "", "", "", fmt.Errorf("could not find '%s' section in file", TodosHeader)
	}

	// Calculate the end of the header
	headerEndIndex := todosHeaderIndex + len(TodosHeader)
	if headerEndIndex >= len(content) {
		return "", "", "", fmt.Errorf("incomplete %s section: no content after header", TodosHeader)
	}

	contentAfterHeader := content[headerEndIndex:]

	// Find the first blank line after the header
	blankLineIndex := strings.Index(contentAfterHeader, BlankLineSeparator)
	if blankLineIndex == -1 {
		return "", "", "", fmt.Errorf("invalid %s section format: expected blank line after header", TodosHeader)
	}

	// Calculate section boundaries
	beforeTodosEnd := headerEndIndex + blankLineIndex + len(BlankLineSeparator)
	beforeTodos := content[:beforeTodosEnd]

	// Find the next section header (if any)
	afterHeaderContent := content[beforeTodosEnd:]
	nextSectionMatch := NextSectionRegex.FindStringIndex(afterHeaderContent)

	var todosSection string
	var afterTodos string

	if nextSectionMatch != nil {
		// There is another section after Todos
		todosEndIndex := beforeTodosEnd + nextSectionMatch[0]
		todosSection = content[beforeTodosEnd:todosEndIndex]
		afterTodos = content[todosEndIndex:]
	} else {
		// Todos is the last section
		todosSection = afterHeaderContent
		afterTodos = ""
	}

	return beforeTodos, strings.TrimSpace(todosSection), afterTodos, nil
}

// ProcessTodosSection processes the Todos section and returns the completed and uncompleted sections.
// It parses the todos, splits them into completed/uncompleted, adds date tags, and converts back to strings.
// If there are no completed tasks, it returns a "Moved to [[date]]" message for the completed section.
func ProcessTodosSection(todosSection string, originalDate string, currentDate string) (string, string, error) {
	// Validate inputs
	if err := validateProcessInputs(originalDate, currentDate); err != nil {
		return "", "", err
	}

	// Handle empty todos section
	if strings.TrimSpace(todosSection) == "" {
		return fmt.Sprintf(MovedToTemplate, currentDate), "", nil
	}

	// Parse the Todos section into a structured format
	journal, err := ParseTodosSection(todosSection)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse todos section: %w", err)
	}

	// Split the journal into completed and uncompleted tasks
	completedJournal, uncompletedJournal := SplitJournal(journal)

	// Add date tags to completed tasks
	TagCompletedItems(completedJournal, originalDate)

	// Add date tags to completed subtasks in uncompleted tasks
	TagCompletedSubitems(uncompletedJournal, originalDate)

	// Convert back to string format
	completedTodos := JournalToString(completedJournal)
	uncompletedTodos := JournalToString(uncompletedJournal)

	// If there are no completed tasks, show "Moved to [[date]]"
	if completedJournal.IsEmpty() {
		completedTodos = fmt.Sprintf(MovedToTemplate, currentDate)
	}

	return completedTodos, uncompletedTodos, nil
}

// validateProcessInputs validates the inputs for ProcessTodosSection
func validateProcessInputs(originalDate, currentDate string) error {
	if originalDate == "" {
		return fmt.Errorf("original date cannot be empty")
	}
	if currentDate == "" {
		return fmt.Errorf("current date cannot be empty")
	}
	if err := ValidateDate(originalDate); err != nil {
		return fmt.Errorf("invalid original date: %w", err)
	}
	if err := ValidateDate(currentDate); err != nil {
		return fmt.Errorf("invalid current date: %w", err)
	}
	return nil
}

// CreateFromTemplateContent creates file content from template content using Go template syntax.
// It validates inputs, executes the template with the provided data, and cleans up formatting.
// The template receives TemplateData with comprehensive date formatting and todo variables.
func CreateFromTemplateContent(templateContent, todosContent, currentDate, previousDate string) (string, error) {
	// Validate inputs
	if err := validateTemplateInputs(templateContent, currentDate); err != nil {
		return "", err
	}

	// Format current date variables
	currentDateVars := FormatDateVariables(currentDate)

	// Format previous date variables
	previousDateVars := FormatDateVariables(previousDate)

	// Create template data with all date variants
	data := TemplateData{
		Date:         currentDate,
		TODOS:        todosContent,
		PreviousDate: previousDate,

		// Current date variants
		DateShort:  currentDateVars.Short,
		DateLong:   currentDateVars.Long,
		Year:       currentDateVars.Year,
		Month:      currentDateVars.Month,
		MonthName:  currentDateVars.MonthName,
		Day:        currentDateVars.Day,
		DayName:    currentDateVars.DayName,
		WeekNumber: currentDateVars.WeekNumber,

		// Previous date variants
		PreviousDateShort:  previousDateVars.Short,
		PreviousDateLong:   previousDateVars.Long,
		PreviousYear:       previousDateVars.Year,
		PreviousMonth:      previousDateVars.Month,
		PreviousMonthName:  previousDateVars.MonthName,
		PreviousDay:        previousDateVars.Day,
		PreviousDayName:    previousDateVars.DayName,
		PreviousWeekNumber: previousDateVars.WeekNumber,
	}

	// Parse and execute the Go template
	output, err := executeTemplate(templateContent, data)
	if err != nil {
		return "", err
	}

	// Clean up extra blank lines when TODOS is empty
	if strings.TrimSpace(todosContent) == "" {
		output = cleanExcessiveBlankLines(output)
	}

	return output, nil
}

// validateTemplateInputs validates inputs for CreateFromTemplateContent
func validateTemplateInputs(templateContent, currentDate string) error {
	if templateContent == "" {
		return fmt.Errorf("template content cannot be empty")
	}
	if err := ValidateDate(currentDate); err != nil {
		return fmt.Errorf("invalid current date: %w", err)
	}
	return nil
}

// executeTemplate parses and executes a Go template with the provided data
func executeTemplate(templateContent string, data TemplateData) (string, error) {
	tmpl, err := template.New("journal").Funcs(CreateTemplateFunctions()).Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return result.String(), nil
}

// cleanExcessiveBlankLines removes sequences of 3 or more newlines and replaces them with 2 newlines.
// This prevents excessive whitespace when template sections are empty.
func cleanExcessiveBlankLines(content string) string {
	return excessiveBlankLinesRegex.ReplaceAllString(content, BlankLineSeparator)
}

// CreateFromTemplateContentWithStats creates file content from template content using Go template syntax with todo statistics.
// It validates inputs, calculates todo statistics, executes the template with comprehensive data, and cleans up formatting.
// The template receives TemplateData with date formatting, todo statistics, and content variables.
func CreateFromTemplateContentWithStats(templateContent, todosContent, currentDate, previousDate string, journal *TodoJournal) (string, error) {
	// Validate inputs
	if err := validateTemplateInputs(templateContent, currentDate); err != nil {
		return "", err
	}

	// Format current date variables
	currentDateVars := FormatDateVariables(currentDate)

	// Format previous date variables
	previousDateVars := FormatDateVariables(previousDate)

	// Calculate todo statistics
	todoStats := CalculateTodoStatistics(journal, currentDate)

	// Create template data with all variants and statistics
	data := TemplateData{
		Date:         currentDate,
		TODOS:        todosContent,
		PreviousDate: previousDate,

		// Current date variants
		DateShort:  currentDateVars.Short,
		DateLong:   currentDateVars.Long,
		Year:       currentDateVars.Year,
		Month:      currentDateVars.Month,
		MonthName:  currentDateVars.MonthName,
		Day:        currentDateVars.Day,
		DayName:    currentDateVars.DayName,
		WeekNumber: currentDateVars.WeekNumber,

		// Previous date variants
		PreviousDateShort:  previousDateVars.Short,
		PreviousDateLong:   previousDateVars.Long,
		PreviousYear:       previousDateVars.Year,
		PreviousMonth:      previousDateVars.Month,
		PreviousMonthName:  previousDateVars.MonthName,
		PreviousDay:        previousDateVars.Day,
		PreviousDayName:    previousDateVars.DayName,
		PreviousWeekNumber: previousDateVars.WeekNumber,

		// Todo statistics
		TotalTodos:     todoStats.TotalTodos,
		CompletedTodos: todoStats.CompletedTodos,
		UncompletedTodos: todoStats.UncompletedTodos,
		TodoDates:      todoStats.TodoDates,
		OldestTodoDate: todoStats.OldestTodoDate,
		TodoDaysSpan:   todoStats.TodoDaysSpan,
	}

	// Parse and execute the Go template
	output, err := executeTemplate(templateContent, data)
	if err != nil {
		return "", err
	}

	// Clean up extra blank lines when TODOS is empty
	if strings.TrimSpace(todosContent) == "" {
		output = cleanExcessiveBlankLines(output)
	}

	return output, nil
}

// ProcessTodosSectionWithStats processes the Todos section and returns completed/uncompleted sections plus parsed journal.
// Similar to ProcessTodosSection but also returns the original parsed journal for statistics calculation.
func ProcessTodosSectionWithStats(todosSection string, originalDate string, currentDate string) (string, string, *TodoJournal, error) {
	// Validate inputs
	if err := validateProcessInputs(originalDate, currentDate); err != nil {
		return "", "", nil, err
	}

	// Handle empty todos section
	if strings.TrimSpace(todosSection) == "" {
		return fmt.Sprintf(MovedToTemplate, currentDate), "", &TodoJournal{}, nil
	}

	// Parse the Todos section into a structured format
	journal, err := ParseTodosSection(todosSection)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to parse todos section: %w", err)
	}

	// Split the journal into completed and uncompleted tasks
	completedJournal, uncompletedJournal := SplitJournal(journal)

	// Add date tags to completed tasks
	TagCompletedItems(completedJournal, originalDate)

	// Add date tags to completed subtasks in uncompleted tasks
	TagCompletedSubitems(uncompletedJournal, originalDate)

	// Convert back to string format
	completedSection := JournalToString(completedJournal)
	uncompletedSection := JournalToString(uncompletedJournal)

	// If no completed tasks, provide moved message
	if strings.TrimSpace(completedSection) == "" {
		completedSection = fmt.Sprintf(MovedToTemplate, currentDate)
	}

	// Return original journal for statistics calculation
	return completedSection, uncompletedSection, journal, nil
}

// CreateFromTemplateContentWithCustom creates template output with comprehensive data including custom variables.
// This is the most advanced template rendering function that supports date formatting, todo statistics, and custom variables.
func CreateFromTemplateContentWithCustom(templateContent, todosContent, currentDate, previousDate string, journal *TodoJournal, customVars map[string]interface{}) (string, error) {
	// Validate inputs
	if err := validateTemplateInputs(templateContent, currentDate); err != nil {
		return "", err
	}

	// Validate custom variables
	if err := ValidateCustomVariables(customVars); err != nil {
		return "", fmt.Errorf("invalid custom variables: %w", err)
	}

	// Format current date variables
	currentDateVars := FormatDateVariables(currentDate)

	// Format previous date variables
	previousDateVars := FormatDateVariables(previousDate)

	// Calculate todo statistics
	todoStats := CalculateTodoStatistics(journal, currentDate)

	// Create template data with all variants and statistics
	data := TemplateData{
		Date:         currentDate,
		TODOS:        todosContent,
		PreviousDate: previousDate,

		// Current date variants
		DateShort:  currentDateVars.Short,
		DateLong:   currentDateVars.Long,
		Year:       currentDateVars.Year,
		Month:      currentDateVars.Month,
		MonthName:  currentDateVars.MonthName,
		Day:        currentDateVars.Day,
		DayName:    currentDateVars.DayName,
		WeekNumber: currentDateVars.WeekNumber,

		// Previous date variants
		PreviousDateShort:  previousDateVars.Short,
		PreviousDateLong:   previousDateVars.Long,
		PreviousYear:       previousDateVars.Year,
		PreviousMonth:      previousDateVars.Month,
		PreviousMonthName:  previousDateVars.MonthName,
		PreviousDay:        previousDateVars.Day,
		PreviousDayName:    previousDateVars.DayName,
		PreviousWeekNumber: previousDateVars.WeekNumber,

		// Todo statistics
		TotalTodos:     todoStats.TotalTodos,
		CompletedTodos: todoStats.CompletedTodos,
		UncompletedTodos: todoStats.UncompletedTodos,
		UncompletedTopLevelTodos: todoStats.UncompletedTopLevelTodos,
		TodoDates:      todoStats.TodoDates,
		OldestTodoDate: todoStats.OldestTodoDate,
		TodoDaysSpan:   todoStats.TodoDaysSpan,
	}

	// Merge custom variables
	MergeCustomVariables(&data, customVars)

	// Parse and execute the Go template
	output, err := executeTemplate(templateContent, data)
	if err != nil {
		return "", err
	}

	// Clean up extra blank lines when TODOS is empty
	if strings.TrimSpace(todosContent) == "" {
		output = cleanExcessiveBlankLines(output)
	}

	return output, nil
}

// CreateTemplateFunctions returns a map of custom template functions for enhanced template functionality.
// These functions provide date arithmetic, string manipulation, and utility operations for templates.
func CreateTemplateFunctions() template.FuncMap {
	return template.FuncMap{
		// Date arithmetic functions
		"addDays": func(dateStr string, days int) string {
			date, err := time.Parse(DateFormat, dateStr)
			if err != nil {
				return dateStr // Return original on error
			}
			return date.AddDate(0, 0, days).Format(DateFormat)
		},
		"subDays": func(dateStr string, days int) string {
			date, err := time.Parse(DateFormat, dateStr)
			if err != nil {
				return dateStr // Return original on error
			}
			return date.AddDate(0, 0, -days).Format(DateFormat)
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

		// String manipulation functions
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
		"trim": strings.TrimSpace,
		"replace": func(old, new, str string) string {
			return strings.ReplaceAll(str, old, new)
		},
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"split": func(sep, str string) []string {
			return strings.Split(str, sep)
		},
		"join": func(sep string, strs []string) string {
			return strings.Join(strs, sep)
		},
		"repeat": strings.Repeat,
		"len": func(s string) int {
			return len(s)
		},

		// Utility functions
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
			rand.Seed(time.Now().UnixNano())
			for i := len(shuffled) - 1; i > 0; i-- {
				j := rand.Intn(i + 1)
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
			rand.Seed(time.Now().UnixNano())
			for i := len(shuffled) - 1; i > 0; i-- {
				j := rand.Intn(i + 1)
				shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
			}

			return shuffled
		},

		// Simple arithmetic functions for templates
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

		// Day of week functions
		"isMonday": func(dateStr string) bool {
			date, err := time.Parse(DateFormat, dateStr)
			if err != nil {
				return false
			}
			return date.Weekday() == time.Monday
		},
		"isTuesday": func(dateStr string) bool {
			date, err := time.Parse(DateFormat, dateStr)
			if err != nil {
				return false
			}
			return date.Weekday() == time.Tuesday
		},
		"isWednesday": func(dateStr string) bool {
			date, err := time.Parse(DateFormat, dateStr)
			if err != nil {
				return false
			}
			return date.Weekday() == time.Wednesday
		},
		"isThursday": func(dateStr string) bool {
			date, err := time.Parse(DateFormat, dateStr)
			if err != nil {
				return false
			}
			return date.Weekday() == time.Thursday
		},
		"isFriday": func(dateStr string) bool {
			date, err := time.Parse(DateFormat, dateStr)
			if err != nil {
				return false
			}
			return date.Weekday() == time.Friday
		},
		"isSaturday": func(dateStr string) bool {
			date, err := time.Parse(DateFormat, dateStr)
			if err != nil {
				return false
			}
			return date.Weekday() == time.Saturday
		},
		"isSunday": func(dateStr string) bool {
			date, err := time.Parse(DateFormat, dateStr)
			if err != nil {
				return false
			}
			return date.Weekday() == time.Sunday
		},
	}
}
