package generator

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inful/todoer/pkg/core"
)

// TestNewGeneratorWithOptions_Basic tests basic generator creation using options-based constructor
func TestNewGeneratorWithOptions_Basic(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		templateDate string
		wantErr      bool
	}{
		{
			name:         "valid template and date",
			template:     "# {{.Date}}\n{{.TODOS}}",
			templateDate: "2024-01-15",
			wantErr:      false,
		},
		{
			name:         "invalid template date",
			template:     "# {{.Date}}\n{{.TODOS}}",
			templateDate: "invalid-date",
			wantErr:      true,
		},
		{
			name:         "invalid template syntax",
			template:     "# {{.Date}\n{{.TODOS}}",
			templateDate: "2024-01-15",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewGeneratorWithOptions(tt.template, tt.templateDate)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGeneratorWithOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && gen == nil {
				t.Errorf("NewGeneratorWithOptions() returned nil generator")
			}
		})
	}
}

// TestNewGeneratorWithOptions tests the modern options-based constructor
func TestNewGeneratorWithOptions(t *testing.T) {
	template := "# {{.Date}}\n{{.TODOS}}\n{{if .PreviousDate}}Previous: {{.PreviousDate}}{{end}}"
	templateDate := "2024-01-15"

	t.Run("basic options", func(t *testing.T) {
		gen, err := NewGeneratorWithOptions(template, templateDate)
		if err != nil {
			t.Fatalf("NewGeneratorWithOptions() error = %v", err)
		}
		if gen == nil {
			t.Fatal("NewGeneratorWithOptions() returned nil")
		}
	})

	t.Run("with previous date", func(t *testing.T) {
		gen, err := NewGeneratorWithOptions(template, templateDate, WithPreviousDate("2024-01-14"))
		if err != nil {
			t.Fatalf("NewGeneratorWithOptions() error = %v", err)
		}
		if gen.previousDate != "2024-01-14" {
			t.Errorf("previousDate = %v, want %v", gen.previousDate, "2024-01-14")
		}
	})

	t.Run("with custom variables", func(t *testing.T) {
		customVars := map[string]interface{}{
			"projectName": "TodoApp",
			"version":     "1.0.0",
		}
		gen, err := NewGeneratorWithOptions(template, templateDate, WithCustomVariables(customVars))
		if err != nil {
			t.Fatalf("NewGeneratorWithOptions() error = %v", err)
		}
		if gen.customVars["projectName"] != "TodoApp" {
			t.Errorf("customVars[projectName] = %v, want %v", gen.customVars["projectName"], "TodoApp")
		}
	})

	t.Run("invalid custom variables", func(t *testing.T) {
		invalidVars := map[string]interface{}{
			"Date": "reserved name", // Date is reserved
		}
		_, err := NewGeneratorWithOptions(template, templateDate, WithCustomVariables(invalidVars))
		if err == nil {
			t.Error("NewGeneratorWithOptions() should fail with reserved variable name")
		}
	})
}

// TestNewGeneratorFromFile tests file-based generator creation
func TestNewGeneratorFromFile(t *testing.T) {
	// Create a temporary template file
	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "test_template.md")
	templateContent := "# {{.Date}}\n\n## Todos\n\n{{.TODOS}}\n"

	err := os.WriteFile(templateFile, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test template file: %v", err)
	}

	t.Run("valid file", func(t *testing.T) {
		gen, err := NewGeneratorFromFileWithOptions(templateFile, "2024-01-15")
		if err != nil {
			t.Fatalf("NewGeneratorFromFile() error = %v", err)
		}
		if gen.templateContent != templateContent {
			t.Errorf("templateContent mismatch")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := NewGeneratorFromFileWithOptions("nonexistent.md", "2024-01-15")
		if err == nil {
			t.Error("NewGeneratorFromFile() should fail for nonexistent file")
		}
	})
}

// TestNewGeneratorFromFileWithOptions tests file-based generator with options
func TestNewGeneratorFromFileWithOptions(t *testing.T) {
	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "test_template.md")
	templateContent := "# {{.Date}}\n{{if .PreviousDate}}Previous: {{.PreviousDate}}{{end}}\n{{.TODOS}}\n"

	err := os.WriteFile(templateFile, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test template file: %v", err)
	}

	customVars := map[string]interface{}{
		"author": "Test User",
	}

	gen, err := NewGeneratorFromFileWithOptions(templateFile, "2024-01-15",
		WithPreviousDate("2024-01-14"),
		WithCustomVariables(customVars))

	if err != nil {
		t.Fatalf("NewGeneratorFromFileWithOptions() error = %v", err)
	}

	if gen.previousDate != "2024-01-14" {
		t.Errorf("previousDate = %v, want %v", gen.previousDate, "2024-01-14")
	}

	if gen.customVars["author"] != "Test User" {
		t.Errorf("customVars[author] = %v, want %v", gen.customVars["author"], "Test User")
	}
}

// TestGeneratorProcess tests the core processing functionality
func TestGeneratorProcess(t *testing.T) {
	template := `---
title: {{.Date}}
---

# Daily Journal {{.Date}}

## Todos

{{.TODOS}}

## Notes

Notes for the day.`

	gen, err := NewGeneratorWithOptions(template, "2024-01-16")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	originalContent := `---
title: 2024-01-15
---

# Daily Journal 2024-01-15

## Todos

- [[2024-01-14]]
  - [ ] Review code
  - [x] Write tests

- [ ] Plan next sprint
- [x] Complete documentation

## Notes

Today's notes here.`

	result, err := gen.Process(originalContent)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Read the modified original
	modifiedBytes, err := io.ReadAll(result.ModifiedOriginal)
	if err != nil {
		t.Fatalf("Failed to read modified content: %v", err)
	}
	modifiedContent := string(modifiedBytes)

	// Check that completed items are marked with completion date
	if !strings.Contains(modifiedContent, "[x] Write tests #2024-01-15") {
		t.Error("Completed items should be tagged with completion date")
	}

	// Read the new file content
	newBytes, err := io.ReadAll(result.NewFile)
	if err != nil {
		t.Fatalf("Failed to read new file content: %v", err)
	}
	newContent := string(newBytes)

	// Check that new file contains uncompleted tasks
	if !strings.Contains(newContent, "[ ] Review code") {
		t.Error("New file should contain uncompleted tasks")
	}

	// Check that new file uses the template date
	if !strings.Contains(newContent, "title: 2024-01-16") {
		t.Error("New file should use template date")
	}
}

// TestGeneratorProcessFile tests file-based processing
func TestGeneratorProcessFile(t *testing.T) {
	template := "# {{.Date}}\n\n## Todos\n\n{{.TODOS}}\n"
	gen, err := NewGeneratorWithOptions(template, "2024-01-16")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Create a test input file
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "input.md")
	content := `---
title: 2024-01-15
---

## Todos

- [ ] Task 1
- [x] Task 2`

	err = os.WriteFile(inputFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err := gen.ProcessFile(inputFile)
	if err != nil {
		t.Fatalf("ProcessFile() error = %v", err)
	}

	if result == nil {
		t.Fatal("ProcessFile() returned nil result")
	}

	// Verify we can read both results
	_, err = io.ReadAll(result.ModifiedOriginal)
	if err != nil {
		t.Errorf("Failed to read modified original: %v", err)
	}

	_, err = io.ReadAll(result.NewFile)
	if err != nil {
		t.Errorf("Failed to read new file: %v", err)
	}
}

// TestGeneratorWithOptions tests the WithOptions method for reconfiguration
func TestGeneratorWithOptions(t *testing.T) {
	template := "# {{.Date}}\n{{if .PreviousDate}}Previous: {{.PreviousDate}}{{end}}\n{{.TODOS}}\n"

	gen, err := NewGeneratorWithOptions(template, "2024-01-15")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Reconfigure with new options
	newGen, err := gen.WithOptions(
		WithPreviousDate("2024-01-14"),
		WithCustomVariables(map[string]interface{}{
			"author": "Test User",
		}))

	if err != nil {
		t.Fatalf("WithOptions() error = %v", err)
	}

	if newGen.previousDate != "2024-01-14" {
		t.Errorf("WithOptions() previousDate = %v, want %v", newGen.previousDate, "2024-01-14")
	}

	if newGen.customVars["author"] != "Test User" {
		t.Errorf("WithOptions() customVars[author] = %v, want %v", newGen.customVars["author"], "Test User")
	}

	// Original generator should be unchanged
	if gen.previousDate != "" {
		t.Errorf("Original generator should be unchanged")
	}

	// Test that the reconfigured generator can process content
	content := `---
title: 2024-01-14
---

## Todos

- [ ] Task 1`

	result, err := newGen.Process(content)
	if err != nil {
		t.Fatalf("Failed to process with newGen: %v", err)
	}

	if _, err := io.ReadAll(result.ModifiedOriginal); err != nil {
		t.Fatalf("Failed to read modified original: %v", err)
	}
	if _, err := io.ReadAll(result.NewFile); err != nil {
		t.Fatalf("Failed to read new file: %v", err)
	}
}

// TestGeneratorEdgeCases tests edge cases and error conditions
func TestGeneratorEdgeCases(t *testing.T) {
	template := "# {{.Date}}\n{{.TODOS}}\n"

	t.Run("process content without todos section", func(t *testing.T) {
		gen, err := NewGeneratorWithOptions(template, "2024-01-15")
		if err != nil {
			t.Fatalf("Failed to create generator: %v", err)
		}

		content := `---
title: 2024-01-14
---

# Daily Journal

Some content without todos section.`

		result, err := gen.Process(content)
		if err != nil {
			t.Fatalf("Process() should handle missing todos section: %v", err)
		}

		// Should still produce results
		if result == nil {
			t.Fatal("Process() should return result even without todos section")
		}
	})

	t.Run("process empty content", func(t *testing.T) {
		gen, err := NewGeneratorWithOptions(template, "2024-01-15")
		if err != nil {
			t.Fatalf("Failed to create generator: %v", err)
		}

		_, err = gen.Process("")
		if err == nil {
			t.Error("Process() should fail with empty content")
		}
	})

	t.Run("process nonexistent file", func(t *testing.T) {
		gen, err := NewGeneratorWithOptions(template, "2024-01-15")
		if err != nil {
			t.Fatalf("Failed to create generator: %v", err)
		}

		_, err = gen.ProcessFile("nonexistent.md")
		if err == nil {
			t.Error("ProcessFile() should fail for nonexistent file")
		}
	})
}

// TestTemplateValidation tests the template validation functionality
func TestTemplateValidation(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{
			name:     "valid template",
			template: "# {{.Date}}\n{{.TODOS}}",
			wantErr:  false,
		},
		{
			name:     "template with custom functions",
			template: "# {{.Date | formatDate \"2006-01-02\"}}\n{{.TODOS}}",
			wantErr:  false,
		},
		{
			name:     "invalid template syntax - missing closing brace",
			template: "# {{.Date}\n{{.TODOS}}",
			wantErr:  true,
		},
		{
			name:     "invalid template syntax - unknown function",
			template: "# {{.Date | unknownFunction}}\n{{.TODOS}}",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGeneratorWithOptions(tt.template, "2024-01-15")
			if (err != nil) != tt.wantErr {
				t.Errorf("Template validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Deprecated constructor tests removed; NewGeneratorWithOptions/NewGeneratorFromFileWithOptions are the supported APIs.

// TestConvenienceFunctions tests the convenience functions
func TestConvenienceFunctions(t *testing.T) {
	t.Run("ExtractDateFromFrontmatter", func(t *testing.T) {
		content := `---
title: 2024-01-15
---

# Test`
		date, err := ExtractDateFromFrontmatter(content, "title")
		if err != nil {
			t.Errorf("ExtractDateFromFrontmatter() error = %v", err)
		}
		if date != "2024-01-15" {
			t.Errorf("ExtractDateFromFrontmatter() = %v, want %v", date, "2024-01-15")
		}
	})

	t.Run("CreateFromTemplateContent", func(t *testing.T) {
		template := "# {{.Date}}\n{{.TODOS}}"
		todos := "- [ ] Task 1"
		date := "2024-01-15"

		result, err := core.CreateFromTemplate(core.TemplateOptions{
			Content:      template,
			TodosContent: todos,
			CurrentDate:  date,
		})
		if err != nil {
			t.Errorf("CreateFromTemplateContent() error = %v", err)
		}
		if !strings.Contains(result, "2024-01-15") {
			t.Error("Result should contain the date")
		}
		if !strings.Contains(result, "Task 1") {
			t.Error("Result should contain the todos")
		}
	})
}

// BenchmarkGeneratorCreation benchmarks generator creation
func BenchmarkGeneratorCreation(b *testing.B) {
	template := `---
title: {{.Date}}
---

# Daily Journal {{.Date}}

## Todos

{{.TODOS}}

## Notes

Notes for the day.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewGeneratorWithOptions(template, "2024-01-15")
		if err != nil {
			b.Fatalf("Generator creation failed: %v", err)
		}
	}
}

// BenchmarkProcessing benchmarks the processing functionality
func BenchmarkProcessing(b *testing.B) {
	template := "# {{.Date}}\n{{.TODOS}}"
	gen, err := NewGeneratorWithOptions(template, "2024-01-16")
	if err != nil {
		b.Fatalf("Failed to create generator: %v", err)
	}

	content := `---
title: 2024-01-15
---

## Todos

- [ ] Task 1
- [x] Task 2
- [ ] Task 3
- [x] Task 4`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := gen.Process(content)
		if err != nil {
			b.Fatalf("Processing failed: %v", err)
		}
		// Consume the readers to ensure full processing
		if _, err := io.ReadAll(result.ModifiedOriginal); err != nil {
			b.Fatalf("Failed to read modified original: %v", err)
		}
		if _, err := io.ReadAll(result.NewFile); err != nil {
			b.Fatalf("Failed to read new file: %v", err)
		}
	}
}
