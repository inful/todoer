package main

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestValidateFilePath(t *testing.T) {
	tempDir := setupTempDir(t)

	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "empty path",
			path:        "",
			expectError: true,
		},
		{
			name:        "directory traversal",
			path:        "../../../etc/passwd",
			expectError: true,
		},
		{
			name:        "valid relative path",
			path:        "test.md",
			expectError: false,
		},
		{
			name:        "valid absolute path in temp dir",
			path:        filepath.Join(tempDir, "test.md"),
			expectError: false,
		},
		{
			name:        "path with non-existent parent",
			path:        filepath.Join(tempDir, "subdir/test.md"),
			expectError: false, // Should be valid since parent can potentially be created
		},
		{
			name:        "filename starting with two dots is legitimate",
			path:        filepath.Join(tempDir, "..notes.md"),
			expectError: false, // "..notes.md" is a normal filename, not traversal
		},
		{
			name:        "filename containing two dots in middle is legitimate",
			path:        filepath.Join(tempDir, "somedir..backup", "journal.md"),
			expectError: false, // directories with ".." in the middle are valid on every OS
		},
		{
			name:        "parent traversal with explicit double-dot component",
			path:        tempDir + "/../evil.md",
			expectError: true, // ".." as a full path component is traversal
		},
		{
			name:        "traversal to absolute path",
			path:        "../etc/passwd",
			expectError: true, // already covered, but pinning explicitly
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilePath(tt.path)
			if (err != nil) != tt.expectError {
				t.Errorf("validateFilePath() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateDateFormat(t *testing.T) {
	tests := []struct {
		name        string
		date        string
		expectError bool
	}{
		{
			name:        "empty date",
			date:        "",
			expectError: false,
		},
		{
			name:        "valid date",
			date:        "2024-01-15",
			expectError: false,
		},
		{
			name:        "invalid format",
			date:        "01/15/2024",
			expectError: true,
		},
		{
			name:        "invalid date",
			date:        "2024-13-32",
			expectError: true,
		},
		{
			name:        "incomplete date",
			date:        "2024-01",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDateFormat(tt.date)
			if (err != nil) != tt.expectError {
				t.Errorf("validateDateFormat() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateProcessArgs(t *testing.T) {
	tempDir := setupTempDir(t)

	sourceFile := filepath.Join(tempDir, "source.md")
	targetFile := filepath.Join(tempDir, "target.md")

	tests := []struct {
		name         string
		sourceFile   string
		targetFile   string
		templateDate string
		expectError  bool
	}{
		{
			name:        "valid arguments",
			sourceFile:  sourceFile,
			targetFile:  targetFile,
			expectError: false,
		},
		{
			name:         "valid with template date",
			sourceFile:   sourceFile,
			targetFile:   targetFile,
			templateDate: "2024-01-15",
			expectError:  false,
		},
		{
			name:        "same source and target",
			sourceFile:  sourceFile,
			targetFile:  sourceFile,
			expectError: true,
		},
		{
			name:         "invalid template date",
			sourceFile:   sourceFile,
			targetFile:   targetFile,
			templateDate: "invalid",
			expectError:  true,
		},
		{
			name:        "empty source",
			sourceFile:  "",
			targetFile:  targetFile,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProcessArgs(tt.sourceFile, tt.targetFile, tt.templateDate)
			if (err != nil) != tt.expectError {
				t.Errorf("validateProcessArgs() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tempDir := setupTempDir(t)

	// Create a valid template file for testing
	validTemplateFile := filepath.Join(tempDir, "valid_template.md")
	createTestFile(t, validTemplateFile, "# Test Template\n## Todos\n{{.TODOS}}")

	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorType   error
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorType:   ErrInvalidConfig,
		},
		{
			name: "valid config",
			config: &Config{
				RootDir: tempDir,
			},
			expectError: false,
		},
		{
			name: "empty root dir",
			config: &Config{
				RootDir: "",
			},
			expectError: true,
			errorType:   ErrInvalidConfig,
		},
		{
			name: "valid config with template",
			config: &Config{
				RootDir:      tempDir,
				TemplateFile: validTemplateFile,
			},
			expectError: false,
		},
		{
			name: "config with invalid root dir path",
			config: &Config{
				RootDir: "../../../etc",
			},
			expectError: true,
			errorType:   ErrInvalidPath,
		},
		{
			name: "config with nonexistent template file",
			config: &Config{
				RootDir:      tempDir,
				TemplateFile: filepath.Join(tempDir, "nonexistent.md"),
			},
			expectError: true,
			errorType:   ErrTemplateNotFound,
		},
		{
			name: "config with directory as template file",
			config: &Config{
				RootDir:      tempDir,
				TemplateFile: tempDir, // Directory instead of file
			},
			expectError: true,
			errorType:   ErrInvalidConfig,
		},
		{
			name: "config with valid custom variables",
			config: &Config{
				RootDir: tempDir,
				Custom: map[string]any{
					"author":  "John Doe",
					"project": "Test Project",
					"version": 1,
					"active":  true,
				},
			},
			expectError: false,
		},
		{
			name: "config with reserved custom variable name",
			config: &Config{
				RootDir: tempDir,
				Custom: map[string]any{
					"Date": "invalid", // Reserved name
				},
			},
			expectError: true,
			errorType:   ErrInvalidConfig,
		},
		{
			name: "config with invalid custom variable name",
			config: &Config{
				RootDir: tempDir,
				Custom: map[string]any{
					"123invalid": "value", // Invalid name starting with number
				},
			},
			expectError: true,
			errorType:   ErrInvalidConfig,
		},
		{
			name: "config with unsupported custom variable type",
			config: &Config{
				RootDir: tempDir,
				Custom: map[string]any{
					"complex": complex(1, 2), // Unsupported type
				},
			},
			expectError: true,
			errorType:   ErrInvalidConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.expectError {
				t.Errorf("validateConfig() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if tt.expectError && tt.errorType != nil {
				if !errors.Is(err, tt.errorType) {
					t.Errorf("validateConfig() error type = %T, expected error type %T", err, tt.errorType)
				}
			}
		})
	}
}

func TestValidateCustomVariables(t *testing.T) {
	tests := []struct {
		name        string
		custom      map[string]any
		expectError bool
	}{
		{
			name:        "nil custom variables",
			custom:      nil,
			expectError: false,
		},
		{
			name:        "empty custom variables",
			custom:      map[string]any{},
			expectError: false,
		},
		{
			name: "valid custom variables",
			custom: map[string]any{
				"author":    "John Doe",
				"project":   "Test",
				"version":   1,
				"active":    true,
				"tags":      []string{"test", "demo"},
				"numbers":   []int{1, 2, 3},
				"mixed":     []any{"string", 42, true},
				"_private":  "value",
				"CamelCase": "value",
			},
			expectError: false,
		},
		{
			name: "reserved variable name",
			custom: map[string]any{
				"TODOS": "invalid",
			},
			expectError: true,
		},
		{
			name: "invalid variable name - starts with number",
			custom: map[string]any{
				"123invalid": "value",
			},
			expectError: true,
		},
		{
			name: "invalid variable name - contains special chars",
			custom: map[string]any{
				"test-value": "value",
			},
			expectError: true,
		},
		{
			name: "unsupported type - complex number",
			custom: map[string]any{
				"complex": complex(1, 2),
			},
			expectError: true,
		},
		{
			name: "unsupported type - map",
			custom: map[string]any{
				"nested": map[string]string{"key": "value"},
			},
			expectError: true,
		},
		{
			name: "unsupported type in array",
			custom: map[string]any{
				"mixed": []any{"valid", complex(1, 2)},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCustomVariables(tt.custom)
			if (err != nil) != tt.expectError {
				t.Errorf("validateCustomVariables() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
