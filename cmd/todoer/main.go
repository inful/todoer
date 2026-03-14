package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
)

// resolveTemplate determines the template content and source based on configuration.
// Returns (content, name, error).
func resolveTemplate(templateFile string) (string, string, error) {
	if templateFile != "" {
		content, err := os.ReadFile(templateFile)
		if err != nil {
			return "", "", fmt.Errorf("failed to read template file '%s': %w", templateFile, err)
		}
		return string(content), templateFile, nil
	}

	// Try config directory template
	configHome, err := getConfigDir()
	if err != nil {
		// Fall back to embedded template if can't determine config dir
		return defaultTemplate, "embedded default template", nil
	}

	configTemplate := filepath.Join(configHome, ConfigDirName, TemplateFileName)
	if _, err := os.Stat(configTemplate); err == nil {
		content, err := os.ReadFile(configTemplate)
		if err != nil {
			return "", "", fmt.Errorf("failed to read config template '%s': %w", configTemplate, err)
		}
		return string(content), configTemplate, nil
	}

	// Fall back to embedded template
	return defaultTemplate, "embedded default template", nil
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

// fatalError logs an error message to stderr and exits with code 1.
func fatalError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
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
	case cmdProcess:
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
