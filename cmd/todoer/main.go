package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
)

type cliOptions struct {
	Debug        bool   `help:"Enable debug logging"`
	RootDir      string `help:"Root directory for journals (overrides config/env)"`
	TemplateFile string `help:"Template file override (applies to commands that render/create journals)"`
	PrintPath    bool   `help:"Print resulting journal path to stdout (for composability, where applicable)"`

	Process processCmd `cmd:"" help:"Process a journal file"`
	New     newCmd     `cmd:"new" help:"Create a new daily journal file"`
	Add     addCmd     `cmd:"add <todo-text>..." help:"Add a todo item to today's journal (creates today's file first if needed)"`
	Preview previewCmd `cmd:"preview" help:"Preview rendering of a template file with a sample TODOS section"`
}

type processCmd struct {
	SourceFile   string `arg:"" help:"Input journal file"`
	TargetFile   string `arg:"" help:"Output file for uncompleted tasks"`
	TemplateDate string `help:"Optional date for template rendering (YYYY-MM-DD)"`
}

type newCmd struct{}

type addCmd struct {
	TodoWords []string `arg:"" name:"todo-text" help:"Todo text to add to today's journal"`
}

type previewCmd struct {
	Date        string `help:"Date for template rendering (YYYY-MM-DD, optional, defaults to today)"`
	TodosFile   string `help:"File containing a sample TODOS section to use for preview (optional)"`
	TodosString string `help:"String containing a sample TODOS section to use for preview (optional, overrides --todos-file)"`
	CustomVars  string `help:"Custom variables as JSON string (optional)"`
}

func loggerForCommand(baseLogger *Logger, printPath bool) *Logger {
	if printPath {
		return baseLogger.WithMode(ModeQuiet)
	}
	return baseLogger
}

func sharedPaths(cli *cliOptions, config *Config) (string, string) {
	rootDir := getConfigValue(cli.RootDir, config.RootDir)
	templateFile := getConfigValue(cli.TemplateFile, config.TemplateFile)
	return rootDir, templateFile
}

func (cmd *newCmd) Run(cli *cliOptions, config *Config, baseLogger *Logger) error {
	logger := loggerForCommand(baseLogger, cli.PrintPath)
	logger.Debug("Executing new command")
	rootDir, templateFile := sharedPaths(cli, config)
	return cmdNew(rootDir, templateFile, cli.PrintPath, config, logger)
}

func (cmd *addCmd) Run(cli *cliOptions, config *Config, baseLogger *Logger) error {
	logger := loggerForCommand(baseLogger, cli.PrintPath)
	logger.Debug("Executing add command")
	rootDir, templateFile := sharedPaths(cli, config)
	todoText := strings.Join(cmd.TodoWords, " ")
	return cmdAdd(rootDir, templateFile, todoText, cli.PrintPath, config, logger)
}

func (cmd *processCmd) Run(cli *cliOptions, config *Config, baseLogger *Logger) error {
	logger := loggerForCommand(baseLogger, cli.PrintPath)
	logger.Debug("Executing process command")
	_, templateFile := sharedPaths(cli, config)
	return processJournal(cmd.SourceFile, cmd.TargetFile, templateFile, cmd.TemplateDate, false, cli.PrintPath, config, logger)
}

func (cmd *previewCmd) Run(cli *cliOptions, config *Config, baseLogger *Logger) error {
	logger := loggerForCommand(baseLogger, cli.PrintPath)
	logger.Debug("Executing preview command")
	_, templateFile := sharedPaths(cli, config)
	return cmdPreview(templateFile, cmd.Date, cmd.TodosFile, cmd.TodosString, cmd.CustomVars, config)
}

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

// fatalError logs an error message to stderr and exits with code 1.
func fatalError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}

//go:embed default_template.md
var defaultTemplate string

func main() {
	cli := &cliOptions{}

	ctx := kong.Parse(cli,
		kong.Name("todoer"),
		kong.Description("Process daily journal files, carrying over unfinished tasks in the TODO section."),
		kong.UsageOnError(),
	)

	// Determine output mode and construct logger
	mode := ModeNormal
	if cli.Debug {
		mode = ModeDebug
	}
	baseLogger := NewLogger(mode)

	// Load configuration from file, environment, and defaults
	config, err := loadConfig()
	if err != nil {
		fatalError("Failed to load configuration: %v", err)
	}

	if cli.Debug {
		baseLogger.Debug("Debug logging enabled")
	}

	ctx.Bind(cli, config, baseLogger)
	if err := ctx.Run(); err != nil {
		fatalError("%v", err)
	}
}
