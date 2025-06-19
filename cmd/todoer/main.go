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

	"github.com/alecthomas/kong"
)

// Constants for the application
const (
	FilePermissions = 0644
)

// ProcessCmd defines arguments for the default 'process' command.
type ProcessCmd struct {
	SourceFile   string `arg:"" name:"source_file" help:"Input journal file" type:"existingfile"`
	TargetFile   string `arg:"" name:"target_file" help:"Output file for uncompleted tasks"`
	TemplateFile string `arg:"optional" name:"template_file" help:"Template for creating the target file (optional, will check $XDG_CONFIG_HOME/todoer/template.md or use embedded default)"`
	TemplateDate string `arg:"optional" name:"template_date" help:"Optional date for template rendering (YYYY-MM-DD)"`
}

// NewCmd defines arguments for the 'new' command.
type NewCmd struct {
	RootDir      string `help:"Root directory for journals" default:"."`
	TemplateFile string `help:"Template for creating the target file (optional, will check $XDG_CONFIG_HOME/todoer/template.md or use embedded default)"`
}

// CLI defines the command-line arguments structure for kong
var CLI struct {
	Process struct {
		SourceFile   string `arg:"" help:"Input journal file" type:"existingfile"`
		TargetFile   string `arg:"" help:"Output file for uncompleted tasks"`
		TemplateFile string `help:"Template for creating the target file (optional, will check $XDG_CONFIG_HOME/todoer/template.md or use embedded default)"`
		TemplateDate string `help:"Optional date for template rendering (YYYY-MM-DD)"`
	} `cmd:"" help:"Process a journal file"`

	New struct {
		RootDir      string `help:"Root directory for journals" default:"."`
		TemplateFile string `help:"Template for creating the target file (optional, will check $XDG_CONFIG_HOME/todoer/template.md or use embedded default)"`
	} `cmd:"new" help:"Create a new daily journal file"`
}

//go:embed default_template.md
var defaultTemplate string

func main() {
	ctx := kong.Parse(&CLI,
		kong.Name("todoer"),
		kong.Description("Process daily journal files, carrying over unfinished tasks in the TODO section."),
		kong.UsageOnError(),
	)

	switch ctx.Command() {
	case "new":
		err := cmdNew(CLI.New.RootDir, CLI.New.TemplateFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "process <source-file> <target-file>":
		err := processJournal(CLI.Process.SourceFile, CLI.Process.TargetFile, CLI.Process.TemplateFile, CLI.Process.TemplateDate, false)
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
