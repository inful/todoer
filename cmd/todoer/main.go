package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"todoer/pkg/core"
	"todoer/pkg/generator"
)

// Constants for the application
const (
	FilePermissions = 0644
)

func main() {

	if len(os.Args) < 4 || len(os.Args) > 5 {
		fmt.Println("Usage: todoer <source_file> <target_file> <template_file> [template_date]")
		fmt.Println("       todoer --examples  (run library usage examples)")
		fmt.Println("  source_file:    Input journal file")
		fmt.Println("  target_file:    Output file for uncompleted tasks")
		fmt.Println("  template_file:  Template for creating the target file")
		fmt.Println("  template_date:  Optional date for template rendering (YYYY-MM-DD)")
		fmt.Println("                  If not provided, current date will be used")
		os.Exit(1)
	}

	sourceFile := os.Args[1]
	targetFile := os.Args[2]
	templateFile := os.Args[3]

	var templateDate string
	if len(os.Args) == 5 {
		templateDate = os.Args[4]
		// Validate the template date format
		_, err := time.Parse(core.DateFormat, templateDate)
		if err != nil {
			fmt.Printf("Error: invalid template date format '%s': %v\n", templateDate, err)
			os.Exit(1)
		}
	} else {
		// Use current date if not provided
		templateDate = time.Now().Format(core.DateFormat)
	}

	// Validate that source and target files are different
	if sourceFile == targetFile {
		fmt.Printf("Error: source and target files cannot be the same\n")
		os.Exit(1)
	}

	// Create generator from template file
	gen, err := generator.NewGeneratorFromFile(templateFile, templateDate)
	if err != nil {
		fmt.Printf("Error creating generator from template %s: %v\n", templateFile, err)
		os.Exit(1)
	}

	// Process the source file
	result, err := gen.ProcessFile(sourceFile)
	if err != nil {
		fmt.Printf("Error processing file %s: %v\n", sourceFile, err)
		os.Exit(1)
	}

	// Read the results
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

	// Write the outputs to files
	err = os.WriteFile(sourceFile, modifiedContentBytes, FilePermissions)
	if err != nil {
		fmt.Printf("Error writing to source file %s: %v\n", sourceFile, err)
		os.Exit(1)
	}

	err = os.WriteFile(targetFile, newContentBytes, FilePermissions)
	if err != nil {
		fmt.Printf("Error writing to target file %s: %v\n", targetFile, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully processed journal.\n")
	fmt.Printf("Completed tasks kept in: %s\n", sourceFile)
	fmt.Printf("Uncompleted tasks moved to: %s\n", targetFile)
	fmt.Printf("Created from template: %s (using date: %s)\n", templateFile, templateDate)
}
