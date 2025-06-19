package main

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"todoer/pkg/core"
	"todoer/pkg/generator"

	"github.com/BurntSushi/toml"
	"github.com/alecthomas/kong"
)

// Constants for the application
const (
	FilePermissions = 0644
)

// Config represents the configuration file structure
type Config struct {
	RootDir      string `toml:"root_dir"`
	TemplateFile string `toml:"template_file"`
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

// loadConfigFile loads configuration from the TOML config file
func loadConfigFile(config *Config) error {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = os.Getenv("HOME") + "/.config"
	}
	configPath := filepath.Join(configHome, "todoer", "config.toml")

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
	SourceFile   string `arg:"" name:"source_file" help:"Input journal file" type:"existingfile"`
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
	Process struct {
		SourceFile   string `arg:"" help:"Input journal file" type:"existingfile"`
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
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	ctx := kong.Parse(&CLI,
		kong.Name("todoer"),
		kong.Description("Process daily journal files, carrying over unfinished tasks in the TODO section."),
		kong.UsageOnError(),
	)

	switch ctx.Command() {
	case "new":
		// CLI flags override config/env values
		rootDir := CLI.New.RootDir
		if rootDir == "" {
			rootDir = config.RootDir
		}
		templateFile := CLI.New.TemplateFile
		if templateFile == "" {
			templateFile = config.TemplateFile
		}

		err := cmdNew(rootDir, templateFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "process <source-file> <target-file>":
		// CLI flags override config/env values
		templateFile := CLI.Process.TemplateFile
		if templateFile == "" {
			templateFile = config.TemplateFile
		}

		err := processJournal(CLI.Process.SourceFile, CLI.Process.TargetFile, templateFile, CLI.Process.TemplateDate, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func getGenerator(templateFile, templateDate string) (*generator.Generator, string, error) {
	var gen *generator.Generator
	var err error
	var templateSource string

	if templateDate == "" {
		templateDate = time.Now().Format(core.DateFormat)
	}

	if templateFile != "" {
		gen, err = generator.NewGeneratorFromFile(templateFile, templateDate)
		templateSource = templateFile
	} else {
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			configHome = os.Getenv("HOME") + "/.config"
		}
		configTemplate := filepath.Join(configHome, "todoer", "template.md")
		if _, statErr := os.Stat(configTemplate); statErr == nil {
			gen, err = generator.NewGeneratorFromFile(configTemplate, templateDate)
			templateSource = configTemplate
		}
		if gen == nil {
			gen, err = generator.NewGenerator(defaultTemplate, templateDate)
			templateSource = "embedded default template"
		}
	}
	if err != nil {
		return nil, "", fmt.Errorf("error creating generator from template: %v", err)
	}
	return gen, templateSource, nil
}

func processJournal(sourceFile, targetFile, templateFile, templateDate string, skipBackup bool) error {
	if sourceFile == targetFile {
		return fmt.Errorf("source and target files cannot be the same")
	}

	if templateDate != "" {
		if _, err := time.Parse(core.DateFormat, templateDate); err != nil {
			return fmt.Errorf("invalid template date format '%s': %v", templateDate, err)
		}
	}

	gen, templateSource, err := getGenerator(templateFile, templateDate)
	if err != nil {
		return err
	}

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

	err = os.WriteFile(targetFile, newContentBytes, FilePermissions)
	if err != nil {
		return fmt.Errorf("error writing to target file %s: %v", targetFile, err)
	}

	fmt.Printf("Successfully processed %s -> %s (template: %s)\n", sourceFile, targetFile, templateSource)

	if len(modifiedContentBytes) > 0 && !skipBackup {
		// Create backup of original file (before any modifications)
		backupFile := sourceFile + ".bak"
		originalContentBytes, err := os.ReadFile(sourceFile)
		if err != nil {
			return fmt.Errorf("error reading original file for backup: %v", err)
		}
		err = os.WriteFile(backupFile, originalContentBytes, FilePermissions)
		if err != nil {
			return fmt.Errorf("error creating backup file %s: %v", backupFile, err)
		}

		// Write the modified original content back to the source file
		err = os.WriteFile(sourceFile, modifiedContentBytes, FilePermissions)
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

func cmdNew(rootDir, templateFile string) error {
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
	return processJournal(closest, journalPath, templateFile, today, skipBackup)
}
