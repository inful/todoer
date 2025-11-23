package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"todoer/pkg/core"
	"todoer/pkg/generator"

	"github.com/alecthomas/kong"
)

// templateSource represents different sources of templates
type templateSource struct {
	content string
	name    string
	err     error
}

// resolveTemplate determines the template content and source based on configuration
func resolveTemplate(templateFile string) templateSource {
	if templateFile != "" {
		content, err := os.ReadFile(templateFile)
		if err != nil {
			return templateSource{err: fmt.Errorf("failed to read template file '%s': %w", templateFile, err)}
		}
		return templateSource{content: string(content), name: templateFile}
	}

	// Try config directory template
	configHome, err := getConfigDir()
	if err != nil {
		// Fall back to embedded template if can't determine config dir
		return templateSource{content: defaultTemplate, name: "embedded default template"}
	}

	configTemplate := filepath.Join(configHome, ConfigDirName, TemplateFileName)
	if _, err := os.Stat(configTemplate); err == nil {
		content, err := os.ReadFile(configTemplate)
		if err != nil {
			return templateSource{err: fmt.Errorf("failed to read config template '%s': %w", configTemplate, err)}
		}
		return templateSource{content: string(content), name: configTemplate}
	}

	// Fall back to embedded template
	return templateSource{content: defaultTemplate, name: "embedded default template"}
}

// CLI defines the command-line arguments structure for kong
var CLI struct {
	Debug bool `help:"Enable debug logging"`

	Process struct {
		SourceFile   string `arg:"" help:"Input journal file"`
		TargetFile   string `arg:"" help:"Output file for uncompleted tasks"`
		TemplateFile string `help:"Template for creating the target file (optional, overrides config/env)"`
		TemplateDate string `help:"Optional date for template rendering (YYYY-MM-DD)"`
		PrintPath    bool   `help:"Print the target file path to stdout (for composability)"`
	} `cmd:"" help:"Process a journal file"`

	New struct {
		RootDir      string `help:"Root directory for journals (overrides config/env)"`
		TemplateFile string `help:"Template for creating the target file (optional, overrides config/env)"`
		PrintPath    bool   `help:"Print the created file path to stdout (for composability)"`
	} `cmd:"new" help:"Create a new daily journal file"`

	Preview struct {
		TemplateFile string `help:"Template file to preview (optional, overrides config/env)"`
		Date         string `help:"Date for template rendering (YYYY-MM-DD, optional, defaults to today)"`
		TodosFile    string `help:"File containing a sample TODOS section to use for preview (optional)"`
		TodosString  string `help:"String containing a sample TODOS section to use for preview (optional, overrides --todos-file)"`
		CustomVars   string `help:"Custom variables as JSON string (optional)"`
	} `cmd:"preview" help:"Preview rendering of a template file with a sample TODOS section"`
}

//go:embed default_template.md
var defaultTemplate string

func main() {
	// Determine output mode and construct logger
	mode := ModeNormal
	if CLI.Debug {
		mode = ModeDebug
	}
	// Logger will be further adjusted per-command for quiet mode
	baseLogger := NewLogger(mode)

	// Load configuration from file, environment, and defaults
	config, err := loadConfig()
	if err != nil {
		fatalError("Failed to load configuration: %v", err)
	}

	ctx := kong.Parse(&CLI,
		kong.Name("todoer"),
		kong.Description("Process daily journal files, carrying over unfinished tasks in the TODO section."),
		kong.UsageOnError(),
	)

	if CLI.Debug {
		baseLogger.Debug("Debug logging enabled")
	}

	switch ctx.Command() {
	case "new":
		logger := baseLogger
		if CLI.New.PrintPath {
			logger = logger.WithMode(ModeQuiet)
		}
		logger.Debug("Executing new command")
		rootDir := getConfigValue(CLI.New.RootDir, config.RootDir)
		templateFile := getConfigValue(CLI.New.TemplateFile, config.TemplateFile)

		err := cmdNew(rootDir, templateFile, CLI.New.PrintPath, config, logger)
		if err != nil {
			fatalError("Failed to create new journal: %v", err)
		}
	case "process <source-file> <target-file>":
		logger := baseLogger
		if CLI.Process.PrintPath {
			logger = logger.WithMode(ModeQuiet)
		}
		logger.Debug("Executing process command")
		templateFile := getConfigValue(CLI.Process.TemplateFile, config.TemplateFile)

		err := processJournal(CLI.Process.SourceFile, CLI.Process.TargetFile, templateFile, CLI.Process.TemplateDate, false, CLI.Process.PrintPath, config, logger)
		if err != nil {
			fatalError("Processing failed: %v", err)
		}
	case "preview":
		logger := baseLogger
		logger.Debug("Executing preview command")
		err := cmdPreview(CLI.Preview.TemplateFile, CLI.Preview.Date, CLI.Preview.TodosFile, CLI.Preview.TodosString, CLI.Preview.CustomVars, config)
		if err != nil {
			fatalError("Preview failed: %v", err)
		}
		// Removed: case "completion <shell>":
		// Shell completion is not supported at runtime. See documentation for integration instructions.
	}
}

func getGenerator(templateFile, templateDate, sourceFile string, config *Config) (*generator.Generator, string, error) {
	if templateDate == "" {
		templateDate = time.Now().Format(core.DateFormat)
	}

	// Extract previous date from source file if available
	previousDate := ""
	if sourceFile != "" {
		if content, readErr := os.ReadFile(sourceFile); readErr == nil {
			if extractedDate, extractErr := generator.ExtractDateFromFrontmatter(string(content), config.FrontmatterDateKey); extractErr == nil {
				previousDate = extractedDate
			}
		}
	}

	// Resolve template content and source
	tmplSource := resolveTemplate(templateFile)
	if tmplSource.err != nil {
		return nil, "", fmt.Errorf("error resolving template: %w", tmplSource.err)
	}

	// Create generator with resolved template
	gen, err := generator.NewGeneratorWithOptions(tmplSource.content, templateDate,
		generator.WithPreviousDate(previousDate),
		generator.WithCustomVariables(config.Custom),
		generator.WithFrontmatterDateKey(config.FrontmatterDateKey),
		generator.WithTodosHeader(config.TodosHeader),
	)

	if err != nil {
		return nil, "", fmt.Errorf("error creating generator from template: %w", err)
	}

	return gen, tmplSource.name, nil
}

func processJournal(sourceFile, targetFile, templateFile, templateDate string, skipBackup, printPath bool, config *Config, logger *Logger) error {
	logger.Debug("Processing journal: source=%s, target=%s, template=%s, date=%s", sourceFile, targetFile, templateFile, templateDate)

	quiet := printPath

	// Validate all input arguments
	if err := validateProcessArgs(sourceFile, targetFile, templateDate); err != nil {
		return err
	}

	if err := validateConfig(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	gen, templateSource, err := getGenerator(templateFile, templateDate, sourceFile, config)
	if err != nil {
		return err
	}

	logger.Debug("Using template source: %s", templateSource)

	result, err := gen.ProcessFile(sourceFile)
	if err != nil {
		return fmt.Errorf("error processing file %s: %v", sourceFile, err)
	}

	modifiedContentBytes, err := io.ReadAll(result.ModifiedOriginal)
	if err != nil {
		return fmt.Errorf("error reading modified content: %v", err)
	}

	newContentBytes, err := io.ReadAll(result.NewFile)
	if err != nil {
		return fmt.Errorf("error reading new file content: %v", err)
	}

	logger.Debug("Writing %d bytes to target file: %s", len(newContentBytes), targetFile)
	err = safeWriteFile(targetFile, newContentBytes, FilePermissions)
	if err != nil {
		return fmt.Errorf("error writing to target file %s: %v", targetFile, err)
	}

	logger.Info("Successfully processed %s -> %s (template: %s)", sourceFile, targetFile, templateSource)

	if printPath {
		fmt.Println(targetFile)
	}

	if len(modifiedContentBytes) > 0 && !skipBackup {
		// Create backup of original file (before any modifications)
		backupFile := sourceFile + ".bak"
		originalContentBytes, err := os.ReadFile(sourceFile)
		if err != nil {
			return fmt.Errorf("error reading original file for backup: %v", err)
		}
		err = safeWriteFile(backupFile, originalContentBytes, FilePermissions)
		if err != nil {
			return fmt.Errorf("error creating backup file %s: %v", backupFile, err)
		}

		// Write the modified original content back to the source file
		err = safeWriteFile(sourceFile, modifiedContentBytes, FilePermissions)
		if err != nil {
			return fmt.Errorf("error updating source file %s: %v", sourceFile, err)
		}

		if !quiet {
			fmt.Printf("Backup of original file created: %s\n", backupFile)
		}
	} else {
		if !quiet {
			fmt.Printf("No modifications found in the original file, backup not created.\n")
		}
	}
	return nil
}

func findClosestJournalFile(rootDir, today string) (string, error) {
	var closestFile string
	var minDiff time.Duration = -1

	todayTime, err := time.Parse(core.DateFormat, today)
	if err != nil {
		return "", fmt.Errorf("invalid today date: %w", err)
	}

	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		base := filepath.Base(path)
		// expecting YYYY-MM-DD.md
		if len(base) != len("2006-01-02.md") || filepath.Ext(base) != ".md" {
			return nil
		}

		// Remove .md extension for date parsing
		dateStr := strings.TrimSuffix(base, ".md")
		fileTime, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			// not a journal file
			return nil
		}

		if fileTime.Before(todayTime) {
			diff := todayTime.Sub(fileTime)
			if minDiff == -1 || diff < minDiff {
				minDiff = diff
				closestFile = path
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if closestFile == "" {
		return "", fmt.Errorf("no previous journal found in %s", rootDir)
	}

	return closestFile, nil
}

func cmdNew(rootDir, templateFile string, printPath bool, config *Config, logger *Logger) error {
	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(rootDir, today)

	if _, err := os.Stat(journalPath); err == nil {
		if printPath {
			fmt.Println(journalPath)
		} else {
			fmt.Printf("Journal for today already exists: %s\n", journalPath)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(journalPath), 0755); err != nil {
		return err
	}

	closest, err := findClosestJournalFile(rootDir, today)
	skipBackup := false
	if err != nil {
		if !printPath {
			fmt.Println("No previous journal found, creating a new one from template.")
		}
		// Create an empty temporary file to seed the new journal
		tmpFile, err := os.CreateTemp("", "empty-journal-*.md")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		// The parser needs the `## Todos` header to function correctly.
		if _, err := tmpFile.WriteString(core.TodosHeader + "\n\n"); err != nil {
			return fmt.Errorf("failed to write to temp file: %w", err)
		}
		if err := tmpFile.Close(); err != nil {
			return fmt.Errorf("failed to close temp file: %w", err)
		}

		closest = tmpFile.Name()
		skipBackup = true
	}

	if !printPath {
		fmt.Printf("Using '%s' as source to create new journal for today.\n", closest)
	}

	err = processJournal(closest, journalPath, templateFile, today, skipBackup, printPath, config, logger)
	if err != nil {
		return err
	}

	return nil
}

// buildJournalPath constructs the path for a journal file based on date and root directory
func buildJournalPath(rootDir, date string) string {
	t, err := time.Parse(core.DateFormat, date)
	if err != nil {
		// Fallback to current time if date parsing fails
		t = time.Now()
	}
	year := t.Format("2006")
	month := t.Format("01")
	return filepath.Join(rootDir, year, month, date+".md")
}

// safeWriteFile writes data to a file safely with atomic operations
func safeWriteFile(filename string, data []byte, perm os.FileMode) error {
	// Create a temporary file in the same directory for atomic write
	dir := filepath.Dir(filename)
	tmpFile, err := os.CreateTemp(dir, filepath.Base(filename)+".tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	// Ensure cleanup on any error
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	// Write data to temporary file
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}

	// Close the temporary file
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Atomically move the temporary file to the target location
	if err := os.Rename(tmpFile.Name(), filename); err != nil {
		return fmt.Errorf("failed to move temporary file to target: %w", err)
	}

	// Set correct permissions
	if err := os.Chmod(filename, perm); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// getConfigValue returns the CLI value if provided, otherwise falls back to config value
func getConfigValue(cliValue, configValue string) string {
	if cliValue != "" {
		return cliValue
	}
	return configValue
}

// fatalError logs an error and exits with code 1
func fatalError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}

// cmdPreview renders a template with a sample TODOS section and prints the result to stdout.
func cmdPreview(templateFile, date, todosFile, todosString, customVars string, config *Config) error {
	// Determine date to use
	if date == "" {
		date = time.Now().Format(core.DateFormat)
	}

	// Load sample TODOS section
	var todosContent string
	if todosString != "" {
		todosContent = todosString
	} else if todosFile != "" {
		content, err := os.ReadFile(todosFile)
		if err != nil {
			return fmt.Errorf("failed to read todos file: %w", err)
		}
		todosContent = string(content)
	} else {
		// Built-in default sample TODOS section with multiple days and mixed completion
		todosContent = `- [[2025-06-20]]
  - [ ] Task from Friday
  - [x] Completed Friday task
- [[2025-06-21]]
  - [ ] Task from Saturday
  - [x] Completed Saturday task
    Continuation for completed
  - [ ] Another open Saturday task
    - [ ] Subtask
      - [ ] Sub-subtask
- [[2025-06-22]]
  - [ ] Task from Sunday with #2025-06-22 tag
  - [x] Completed Sunday task`
	}

	// Parse custom variables if provided
	custom := config.Custom
	if customVars != "" {
		parsed, err := parseCustomVarsJSON(customVars)
		if err != nil {
			return fmt.Errorf("failed to parse custom vars: %w", err)
		}
		custom = parsed
	}

	// Resolve template content
	tmplSource := resolveTemplate(templateFile)
	if tmplSource.err != nil {
		return fmt.Errorf("error resolving template: %w", tmplSource.err)
	}

	// Create a dummy journal for statistics
	journal, err := core.ParseTodosSection(todosContent)
	if err != nil {
		return fmt.Errorf("failed to parse todos section: %w", err)
	}

	// Render template
	output, err := core.CreateFromTemplate(core.TemplateOptions{
		Content:      tmplSource.content,
		TodosContent: todosContent,
		CurrentDate:  date,
		PreviousDate: "",
		Journal:      journal,
		CustomVars:   custom,
	})
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Print to stdout
	fmt.Println(output)
	return nil
}

// parseCustomVarsJSON parses a JSON string into a map[string]interface{} for custom variables.
func parseCustomVarsJSON(jsonStr string) (map[string]interface{}, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var m map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
