package main

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"todoer/pkg/core"
	"todoer/pkg/generator"

	"github.com/BurntSushi/toml"
	"github.com/alecthomas/kong"
)

// Validation errors
var (
	ErrInvalidPath      = errors.New("invalid file path")
	ErrSameSourceTarget = errors.New("source and target files cannot be the same")
	ErrInvalidDate      = errors.New("invalid date format")
	ErrInvalidConfig    = errors.New("invalid configuration")
)

// Logger for debug and verbose output
var debugLogger *log.Logger

// Constants for the application
const (
	FilePermissions  = 0644
	ConfigDirName    = "todoer"
	ConfigFileName   = "config.toml"
	TemplateFileName = "template.md"
)

// Config represents the configuration file structure
type Config struct {
	RootDir      string                 `toml:"root_dir"`
	TemplateFile string                 `toml:"template_file"`
	Custom       map[string]interface{} `toml:"custom_variables"`
}

// loadConfig loads configuration from file, environment variables, and CLI flags
// Priority: CLI flags > environment variables > config file > defaults
func loadConfig() (*Config, error) {
	config := &Config{}

	// Load from config file first
	if err := loadConfigFile(config); err != nil {
		// Config file errors are not fatal, just log them
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: Error loading config file: %v\n", err)
		}
	}

	// Override with environment variables
	if rootDir := os.Getenv("TODOER_ROOT_DIR"); rootDir != "" {
		config.RootDir = expandPath(rootDir)
	}
	if templateFile := os.Getenv("TODOER_TEMPLATE_FILE"); templateFile != "" {
		config.TemplateFile = expandPath(templateFile)
	}

	// Set defaults if not specified
	if config.RootDir == "" {
		config.RootDir = "."
	}

	// Validate the final configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// expandPath expands ~ to home directory in file paths
func expandPath(path string) string {
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path // Return original if we can't get home dir
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

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
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = os.Getenv("HOME") + "/.config"
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

// loadConfigFile loads configuration from the TOML config file
func loadConfigFile(config *Config) error {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = os.Getenv("HOME") + "/.config"
	}
	configPath := filepath.Join(configHome, ConfigDirName, ConfigFileName)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return err
	}

	if _, err := toml.DecodeFile(configPath, config); err != nil {
		return fmt.Errorf("failed to decode config file %s: %w", configPath, err)
	}

	// Expand paths that might contain ~
	config.RootDir = expandPath(config.RootDir)
	config.TemplateFile = expandPath(config.TemplateFile)

	return nil
}

// ProcessCmd defines arguments for the default 'process' command.
type ProcessCmd struct {
	SourceFile   string `arg:"" name:"source_file" help:"Input journal file"`
	TargetFile   string `arg:"" name:"target_file" help:"Output file for uncompleted tasks"`
	TemplateFile string `arg:"optional" name:"template_file" help:"Template for creating the target file (optional, overrides config/env)"`
	TemplateDate string `arg:"optional" name:"template_date" help:"Optional date for template rendering (YYYY-MM-DD)"`
}

// NewCmd defines arguments for the 'new' command.
type NewCmd struct {
	RootDir      string `help:"Root directory for journals (overrides config/env)"`
	TemplateFile string `help:"Template for creating the target file (optional, overrides config/env)"`
}

// CLI defines the command-line arguments structure for kong
var CLI struct {
	Debug bool `help:"Enable debug logging"`

	Process struct {
		SourceFile   string `arg:"" help:"Input journal file"`
		TargetFile   string `arg:"" help:"Output file for uncompleted tasks"`
		TemplateFile string `help:"Template for creating the target file (optional, overrides config/env)"`
		TemplateDate string `help:"Optional date for template rendering (YYYY-MM-DD)"`
	} `cmd:"" help:"Process a journal file"`

	New struct {
		RootDir      string `help:"Root directory for journals (overrides config/env)"`
		TemplateFile string `help:"Template for creating the target file (optional, overrides config/env)"`
	} `cmd:"new" help:"Create a new daily journal file"`
}

//go:embed default_template.md
var defaultTemplate string

func main() {
	// Load configuration from file, environment, and defaults
	config, err := loadConfig()
	if err != nil {
		logError("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	ctx := kong.Parse(&CLI,
		kong.Name("todoer"),
		kong.Description("Process daily journal files, carrying over unfinished tasks in the TODO section."),
		kong.UsageOnError(),
	)

	// Enable debug logging if requested
	if CLI.Debug {
		enableDebugLogging()
		logDebug("Debug logging enabled")
	}

	switch ctx.Command() {
	case "new":
		logDebug("Executing new command")
		// CLI flags override config/env values
		rootDir := CLI.New.RootDir
		if rootDir == "" {
			rootDir = config.RootDir
		}
		templateFile := CLI.New.TemplateFile
		if templateFile == "" {
			templateFile = config.TemplateFile
		}

		err := cmdNew(rootDir, templateFile, config)
		if err != nil {
			logError("Failed to create new journal: %v", err)
			os.Exit(1)
		}
	case "process <source-file> <target-file>":
		logDebug("Executing process command")
		// CLI flags override config/env values
		templateFile := CLI.Process.TemplateFile
		if templateFile == "" {
			templateFile = config.TemplateFile
		}

		err := processJournal(CLI.Process.SourceFile, CLI.Process.TargetFile, templateFile, CLI.Process.TemplateDate, false, config)
		if err != nil {
			logError("Processing failed: %v", err)
			os.Exit(1)
		}
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
			if extractedDate, extractErr := generator.ExtractDateFromFrontmatter(string(content)); extractErr == nil {
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
		generator.WithCustomVariables(config.Custom))

	if err != nil {
		return nil, "", fmt.Errorf("error creating generator from template: %w", err)
	}

	return gen, tmplSource.name, nil
}

func processJournal(sourceFile, targetFile, templateFile, templateDate string, skipBackup bool, config *Config) error {
	logDebug("Processing journal: source=%s, target=%s, template=%s, date=%s", sourceFile, targetFile, templateFile, templateDate)

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

	logDebug("Using template source: %s", templateSource)

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

	logDebug("Writing %d bytes to target file: %s", len(newContentBytes), targetFile)
	err = safeWriteFile(targetFile, newContentBytes, FilePermissions)
	if err != nil {
		return fmt.Errorf("error writing to target file %s: %v", targetFile, err)
	}

	logInfo("Successfully processed %s -> %s (template: %s)", sourceFile, targetFile, templateSource)

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

		fmt.Printf("Backup of original file created: %s\n", backupFile)
	} else {
		fmt.Printf("No modifications found in the original file, backup not created.\n")
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

func cmdNew(rootDir, templateFile string, config *Config) error {
	today := time.Now().Format(core.DateFormat)
	month := time.Now().Format("01")
	year := time.Now().Format("2006")
	journalPath := filepath.Join(rootDir, year, month, today+".md")

	if _, err := os.Stat(journalPath); err == nil {
		fmt.Printf("Journal for today already exists: %s\n", journalPath)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(journalPath), 0755); err != nil {
		return err
	}

	closest, err := findClosestJournalFile(rootDir, today)
	skipBackup := false
	if err != nil {
		fmt.Println("No previous journal found, creating a new one from template.")
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

	fmt.Printf("Using '%s' as source to create new journal for today.\n", closest)
	return processJournal(closest, journalPath, templateFile, today, skipBackup, config)
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

// validateFilePath validates a file path for security and correctness
func validateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: path cannot be empty", ErrInvalidPath)
	}

	// Clean the path to resolve any .. or . components
	cleanPath := filepath.Clean(path)

	// Check for potentially dangerous paths
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("%w: path contains directory traversal", ErrInvalidPath)
	}

	// Check if the directory portion exists or can be created
	dir := filepath.Dir(cleanPath)
	if dir != "." && dir != "/" {
		if info, err := os.Stat(dir); err != nil {
			if os.IsNotExist(err) {
				// Check if we can potentially create the directory
				parent := filepath.Dir(dir)
				if parent != dir { // Avoid infinite recursion
					if err := validateFilePath(parent); err != nil {
						return fmt.Errorf("%w: cannot access parent directory", ErrInvalidPath)
					}
				}
			} else {
				return fmt.Errorf("%w: cannot access directory: %v", ErrInvalidPath, err)
			}
		} else if !info.IsDir() {
			return fmt.Errorf("%w: parent path is not a directory", ErrInvalidPath)
		}
	}

	return nil
}

// validateDateFormat validates date string format
func validateDateFormat(date string) error {
	if date == "" {
		return nil // Empty date is valid (will use current date)
	}

	if _, err := time.Parse(core.DateFormat, date); err != nil {
		return fmt.Errorf("%w: expected format YYYY-MM-DD, got %s", ErrInvalidDate, date)
	}

	return nil
}

// validateProcessArgs validates arguments for the process command
func validateProcessArgs(sourceFile, targetFile, templateDate string) error {
	if err := validateFilePath(sourceFile); err != nil {
		return fmt.Errorf("invalid source file: %w", err)
	}

	if err := validateFilePath(targetFile); err != nil {
		return fmt.Errorf("invalid target file: %w", err)
	}

	// Check that source and target are different
	absSource, err := filepath.Abs(sourceFile)
	if err != nil {
		return fmt.Errorf("cannot resolve source file path: %w", err)
	}

	absTarget, err := filepath.Abs(targetFile)
	if err != nil {
		return fmt.Errorf("cannot resolve target file path: %w", err)
	}

	if absSource == absTarget {
		return ErrSameSourceTarget
	}

	if err := validateDateFormat(templateDate); err != nil {
		return fmt.Errorf("invalid template date: %w", err)
	}

	return nil
}

// validateConfig validates the configuration structure
func validateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("%w: config cannot be nil", ErrInvalidConfig)
	}

	if config.RootDir == "" {
		return fmt.Errorf("%w: root directory cannot be empty", ErrInvalidConfig)
	}

	// Validate root directory path
	if err := validateFilePath(config.RootDir); err != nil {
		return fmt.Errorf("invalid root directory: %w", err)
	}

	// Validate template file if specified
	if config.TemplateFile != "" {
		if err := validateFilePath(config.TemplateFile); err != nil {
			return fmt.Errorf("invalid template file: %w", err)
		}
	}

	return nil
}

// enableDebugLogging enables debug logging to stderr
func enableDebugLogging() {
	debugLogger.SetOutput(os.Stderr)
}

// logDebug logs a debug message if debug logging is enabled
func logDebug(format string, args ...interface{}) {
	debugLogger.Printf(format, args...)
}

// logInfo logs an info message to stdout
func logInfo(format string, args ...interface{}) {
	fmt.Printf("INFO: "+format+"\n", args...)
}

// logError logs an error message to stderr
func logError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
}

// init initializes the application, setting up logging and other settings
func init() {
	// Initialize debug logger - disabled by default
	debugLogger = log.New(io.Discard, "DEBUG: ", log.LstdFlags|log.Lshortfile)
}
