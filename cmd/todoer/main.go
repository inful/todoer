package main

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"time"
	"todoer/pkg/core"
	"todoer/pkg/generator"

	"github.com/alecthomas/kong"
)

// Constants for the application
const (
	FilePermissions = 0644
)

// CLI defines the command-line arguments structure for kong
var CLI struct {
	SourceFile   string `arg:"" name:"source_file" help:"Input journal file" type:"existingfile"`
	TargetFile   string `arg:"" name:"target_file" help:"Output file for uncompleted tasks"`
	TemplateFile string `arg:"optional" name:"template_file" help:"Template for creating the target file (optional, will check $XDG_CONFIG_HOME/todoer/template.md or use embedded default)"`
	TemplateDate string `arg:"optional" name:"template_date" help:"Optional date for template rendering (YYYY-MM-DD)"`
}

//go:embed default_template.md
var defaultTemplate string

func main() {
	ctx := kong.Parse(&CLI,
		kong.Name("todoer"),
		kong.Description("Process daily journal files, carrying over unfinished tasks in the TODO section."),
		kong.UsageOnError(),
	)

	if CLI.SourceFile == CLI.TargetFile {
		fmt.Printf("Error: source and target files cannot be the same\n")
		os.Exit(1)
	}

	templateDate := CLI.TemplateDate
	if templateDate == "" {
		templateDate = time.Now().Format(core.DateFormat)
	} else {
		_, err := time.Parse(core.DateFormat, templateDate)
		if err != nil {
			fmt.Printf("Error: invalid template date format '%s': %v\n", templateDate, err)
			os.Exit(1)
		}
	}

	var gen *generator.Generator
	var err error
	var templateSource string

	if CLI.TemplateFile != "" {
		gen, err = generator.NewGeneratorFromFile(CLI.TemplateFile, templateDate)
		templateSource = CLI.TemplateFile
	} else {
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			configHome = os.Getenv("HOME") + "/.config"
		}
		configTemplate := configHome + "/todoer/template.md"
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
		fmt.Printf("Error creating generator from template: %v\n", err)
		os.Exit(1)
	}

	result, err := gen.ProcessFile(CLI.SourceFile)
	if err != nil {
		fmt.Printf("Error processing file %s: %v\n", CLI.SourceFile, err)
		os.Exit(1)
	}

	modifiedContentBytes, err := io.ReadAll(result.ModifiedOriginal)
	if err != nil {
		fmt.Printf("Error reading modified content: %v\n", err)
		os.Exit(1)
	}

	newContentBytes, err := io.ReadAll(result.NewFile)
	if err != nil {
		fmt.Printf("Error reading new file content: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(CLI.TargetFile, newContentBytes, FilePermissions)
	if err != nil {
		fmt.Printf("Error writing to target file %s: %v\n", CLI.TargetFile, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully processed %s -> %s (template: %s)\n", CLI.SourceFile, CLI.TargetFile, templateSource)

	if len(modifiedContentBytes) > 0 {
		backupFile := CLI.SourceFile + ".bak"
		err = os.WriteFile(backupFile, modifiedContentBytes, FilePermissions)
		if err != nil {
			fmt.Printf("Error creating backup file %s: %v\n", backupFile, err)
			os.Exit(1)
		}
		fmt.Printf("Backup of original file created: %s\n", backupFile)
	} else {
		fmt.Printf("No modifications found in the original file, backup not created.\n")
	}
	ctx.Exit(0)
}
