package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Helper function to create a temporary directory and clean it up
func setupTempDir(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "todoer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cleanup := func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to cleanup temp dir: %v", err)
		}
	}

	return tmpDir, cleanup
}

// Helper function to create a test file
func createTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "absolute path",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "relative path",
			input:    "relative/path",
			expected: "relative/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			if tt.name == "home expansion" {
				// For home expansion, just check that ~ is replaced
				if strings.HasPrefix(tt.input, "~/") && strings.Contains(result, "~") {
					t.Errorf("expandPath() = %v, expected ~ to be expanded", result)
				}
			} else {
				if result != tt.expected {
					t.Errorf("expandPath() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestResolveTemplate(t *testing.T) {
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	// Test cases
	tests := []struct {
		name         string
		templateFile string
		setupFunc    func()
		expectError  bool
		expectName   string
	}{
		{
			name:         "empty template file - uses embedded",
			templateFile: "",
			expectError:  false,
			expectName:   "embedded default template",
		},
		{
			name:         "explicit template file - exists",
			templateFile: filepath.Join(tempDir, "custom.md"),
			setupFunc: func() {
				createTestFile(t, filepath.Join(tempDir, "custom.md"), "Custom template content")
			},
			expectError: false,
			expectName:  filepath.Join(tempDir, "custom.md"),
		},
		{
			name:         "explicit template file - missing",
			templateFile: filepath.Join(tempDir, "missing.md"),
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			result := resolveTemplate(tt.templateFile)

			if tt.expectError {
				if result.err == nil {
					t.Errorf("resolveTemplate() expected error, got none")
				}
			} else {
				if result.err != nil {
					t.Errorf("resolveTemplate() unexpected error: %v", result.err)
				}
				if result.name != tt.expectName {
					t.Errorf("resolveTemplate() name = %v, want %v", result.name, tt.expectName)
				}
				if result.content == "" {
					t.Errorf("resolveTemplate() content is empty")
				}
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Save original environment to avoid interference from workspace files
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	originalHome := os.Getenv("HOME")
	originalRootDir := os.Getenv("TODOER_ROOT_DIR")
	originalTemplateFile := os.Getenv("TODOER_TEMPLATE_FILE")

	// Create isolated test environment
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	defer func() {
		// Restore original environment
		if originalXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
		if originalRootDir != "" {
			os.Setenv("TODOER_ROOT_DIR", originalRootDir)
		} else {
			os.Unsetenv("TODOER_ROOT_DIR")
		}
		if originalTemplateFile != "" {
			os.Setenv("TODOER_TEMPLATE_FILE", originalTemplateFile)
		} else {
			os.Unsetenv("TODOER_TEMPLATE_FILE")
		}
	}()

	// Set isolated environment
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	os.Setenv("HOME", tempDir)
	os.Unsetenv("TODOER_ROOT_DIR")
	os.Unsetenv("TODOER_TEMPLATE_FILE")

	// Test loading config with no config file (should succeed with defaults)
	config, err := loadConfig()
	if err != nil {
		t.Errorf("loadConfig() error = %v, want nil", err)
	}
	if config == nil {
		t.Fatalf("loadConfig() returned nil config")
	}
	if config.RootDir == "" {
		t.Errorf("loadConfig() RootDir is empty, expected default")
	}
}

func TestLoadConfigWithXDG(t *testing.T) {
	// Save original environment
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	originalHome := os.Getenv("HOME")
	originalRootDir := os.Getenv("TODOER_ROOT_DIR")
	originalTemplateFile := os.Getenv("TODOER_TEMPLATE_FILE")
	defer func() {
		if originalXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
		if originalRootDir != "" {
			os.Setenv("TODOER_ROOT_DIR", originalRootDir)
		} else {
			os.Unsetenv("TODOER_ROOT_DIR")
		}
		if originalTemplateFile != "" {
			os.Setenv("TODOER_TEMPLATE_FILE", originalTemplateFile)
		} else {
			os.Unsetenv("TODOER_TEMPLATE_FILE")
		}
	}()

	tests := []struct {
		name            string
		setupFunc       func(tempDir string) string // returns expected config path
		xdgConfigHome   string
		expectError     bool
		expectedRootDir string
	}{
		{
			name: "XDG_CONFIG_HOME set with valid config",
			setupFunc: func(tempDir string) string {
				configDir := filepath.Join(tempDir, "todoer")
				if err := os.MkdirAll(configDir, 0755); err != nil {
					t.Fatal(err)
				}
				configPath := filepath.Join(configDir, "config.toml")
				customRoot := filepath.Join(tempDir, "custom_root")
				configContent := fmt.Sprintf(`root_dir = "%s"`, customRoot)
				if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
					t.Fatal(err)
				}
				return configPath
			},
			xdgConfigHome:   "SET_TO_TEMP_DIR", // Special marker to use testTempDir
			expectedRootDir: "DYNAMIC",         // Will be set dynamically to customRoot
		},
		{
			name: "XDG_CONFIG_HOME set but no config file",
			setupFunc: func(tempDir string) string {
				// Create the config directory but no config file
				configDir := filepath.Join(tempDir, "todoer")
				if err := os.MkdirAll(configDir, 0755); err != nil {
					t.Fatal(err)
				}
				return filepath.Join(configDir, "config.toml")
			},
			xdgConfigHome:   "SET_TO_TEMP_DIR", // Special marker to use testTempDir
			expectedRootDir: ".",               // default
		},
		{
			name: "XDG_CONFIG_HOME not set (uses default location)",
			setupFunc: func(tempDir string) string {
				// Create isolated HOME directory to avoid interference
				isolatedHome := filepath.Join(tempDir, "isolated_home")
				if err := os.MkdirAll(isolatedHome, 0755); err != nil {
					t.Fatal(err)
				}
				os.Setenv("HOME", isolatedHome)
				return "" // No config file setup
			},
			expectedRootDir: ".", // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a unique tempDir for this test case
			testTempDir, testCleanup := setupTempDir(t)
			defer testCleanup()

			// Clear environment variables first
			os.Unsetenv("TODOER_ROOT_DIR")
			os.Unsetenv("TODOER_TEMPLATE_FILE")

			// Set XDG_CONFIG_HOME to point to this test's tempDir
			if tt.xdgConfigHome == "SET_TO_TEMP_DIR" {
				os.Setenv("XDG_CONFIG_HOME", testTempDir)
			} else if tt.xdgConfigHome != "" {
				os.Setenv("XDG_CONFIG_HOME", tt.xdgConfigHome)
			} else {
				os.Unsetenv("XDG_CONFIG_HOME")
			}

			// Setup test environment with the test's tempDir
			expectedPath := tt.setupFunc(testTempDir)
			_ = expectedPath // We could use this for additional validation

			// Set dynamic expected root dir if needed
			expectedRootDir := tt.expectedRootDir
			if expectedRootDir == "DYNAMIC" {
				expectedRootDir = filepath.Join(testTempDir, "custom_root")
			}

			// Load config
			config, err := loadConfig()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if config.RootDir != expectedRootDir {
				t.Errorf("Expected RootDir %q, got %q (XDG_CONFIG_HOME=%q)", expectedRootDir, config.RootDir, os.Getenv("XDG_CONFIG_HOME"))
				// Add debug info
				configPath := ""
				if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
					configPath = filepath.Join(xdg, "todoer", "config.toml")
				}
				if _, err := os.Stat(configPath); err == nil {
					t.Errorf("Config file exists at: %s", configPath)
					if content, err := os.ReadFile(configPath); err == nil {
						t.Errorf("Config content: %s", content)
					}
				}
			}
		})
	}
}

func TestResolveTemplateWithXDG(t *testing.T) {
	// Save original environment
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if originalXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	tests := []struct {
		name         string
		setupFunc    func(tempDir string) // Pass tempDir to setup function
		templateFile string
		expectedName string
		expectError  bool
	}{
		{
			name: "XDG_CONFIG_HOME with template file",
			setupFunc: func(tempDir string) {
				configDir := filepath.Join(tempDir, "todoer")
				if err := os.MkdirAll(configDir, 0755); err != nil {
					t.Fatal(err)
				}
				templatePath := filepath.Join(configDir, "template.md")
				templateContent := "# Custom Template\n## Todos\n{{.TODOS}}"
				if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectedName: "DYNAMIC_XDG_TEMPLATE", // Will be set to actual path
		},
		{
			name: "XDG_CONFIG_HOME without template file (falls back to embedded)",
			setupFunc: func(tempDir string) {
				// Don't create template file, but ensure config dir exists
				configDir := filepath.Join(tempDir, "todoer")
				if err := os.MkdirAll(configDir, 0755); err != nil {
					t.Fatal(err)
				}
			},
			expectedName: "embedded default template",
		},
		{
			name: "Explicit template file overrides XDG",
			setupFunc: func(tempDir string) {
				// Create both XDG template and explicit template
				configDir := filepath.Join(tempDir, "todoer")
				if err := os.MkdirAll(configDir, 0755); err != nil {
					t.Fatal(err)
				}
				xdgTemplate := filepath.Join(configDir, "template.md")
				if err := os.WriteFile(xdgTemplate, []byte("XDG Template"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			templateFile: "DYNAMIC_EXPLICIT", // Will be set to explicit.md in tempDir
			expectedName: "DYNAMIC_EXPLICIT", // Will be set to explicit.md in tempDir
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a unique tempDir for this test case
			testTempDir, testCleanup := setupTempDir(t)
			defer testCleanup()

			// Set XDG_CONFIG_HOME to this test's tempDir
			os.Setenv("XDG_CONFIG_HOME", testTempDir)

			// Setup test environment with the test's tempDir
			tt.setupFunc(testTempDir)

			// Set up dynamic values
			var templateFile string
			var expectedName string

			if tt.templateFile == "DYNAMIC_EXPLICIT" {
				templateFile = filepath.Join(testTempDir, "explicit.md")
				expectedName = templateFile
			} else {
				templateFile = tt.templateFile
				expectedName = tt.expectedName
			}

			if expectedName == "DYNAMIC_XDG_TEMPLATE" {
				expectedName = filepath.Join(testTempDir, "todoer", "template.md")
			}

			// Create explicit template file if specified
			if templateFile != "" {
				createTestFile(t, templateFile, "# Explicit Template\n## Todos\n{{.TODOS}}")
			}

			// Resolve template
			result := resolveTemplate(templateFile)

			if tt.expectError {
				if result.err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if result.err != nil {
				t.Errorf("Unexpected error: %v", result.err)
				return
			}

			if result.name != expectedName {
				t.Errorf("Expected template name %q, got %q", expectedName, result.name)
			}

			if result.content == "" {
				t.Error("Template content should not be empty")
			}
		})
	}
}

func TestProcessJournal_ValidationErrors(t *testing.T) {
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	config := &Config{RootDir: tempDir}

	tests := []struct {
		name          string
		sourceFile    string
		targetFile    string
		templateDate  string
		expectError   bool
		errorContains string
	}{
		{
			name:          "same source and target",
			sourceFile:    "same.md",
			targetFile:    "same.md",
			expectError:   true,
			errorContains: "source and target files cannot be the same",
		},
		{
			name:          "invalid template date",
			sourceFile:    "source.md",
			targetFile:    "target.md",
			templateDate:  "invalid-date",
			expectError:   true,
			errorContains: "invalid template date",
		},
		{
			name:        "non-existent source file",
			sourceFile:  filepath.Join(tempDir, "nonexistent.md"),
			targetFile:  filepath.Join(tempDir, "target.md"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processJournal(tt.sourceFile, tt.targetFile, "", tt.templateDate, false, config)

			if tt.expectError {
				if err == nil {
					t.Errorf("processJournal() expected error, got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("processJournal() error = %v, want to contain %v", err, tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("processJournal() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestProcessJournal_Success(t *testing.T) {
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	// Create a valid source file with todos
	sourceContent := `---
date: 2024-01-01
---

# Daily Journal

## Todos

- [ ] Task 1
- [x] Completed task
- [ ] Task 2

## Notes

Some notes here.
`

	sourceFile := filepath.Join(tempDir, "source.md")
	targetFile := filepath.Join(tempDir, "target.md")
	createTestFile(t, sourceFile, sourceContent)

	config := &Config{RootDir: tempDir}

	err := processJournal(sourceFile, targetFile, "", "", false, config)
	if err != nil {
		t.Fatalf("processJournal() unexpected error: %v", err)
	}

	// Check that target file was created
	if _, err := os.Stat(targetFile); err != nil {
		t.Errorf("Target file was not created: %v", err)
	}

	// Check that backup was created
	backupFile := sourceFile + ".bak"
	if _, err := os.Stat(backupFile); err != nil {
		t.Errorf("Backup file was not created: %v", err)
	}
}

func TestFindClosestJournalFile(t *testing.T) {
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	// Create some test journal files
	testFiles := []string{
		"2024/01/2024-01-01.md",
		"2024/01/2024-01-05.md",
		"2024/01/2024-01-10.md",
		"2024/01/other-file.txt", // Should be ignored
	}

	for _, file := range testFiles {
		createTestFile(t, filepath.Join(tempDir, file), "test content")
	}

	tests := []struct {
		name        string
		today       string
		expectFile  string
		expectError bool
	}{
		{
			name:       "find closest before date",
			today:      "2024-01-07",
			expectFile: "2024/01/2024-01-05.md",
		},
		{
			name:       "find closest when multiple exist",
			today:      "2024-01-15",
			expectFile: "2024/01/2024-01-10.md",
		},
		{
			name:        "no previous journals",
			today:       "2024-01-01",
			expectError: true,
		},
		{
			name:        "invalid date format",
			today:       "invalid-date",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := findClosestJournalFile(tempDir, tt.today)

			if tt.expectError {
				if err == nil {
					t.Errorf("findClosestJournalFile() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("findClosestJournalFile() unexpected error: %v", err)
				}
				expectedPath := filepath.Join(tempDir, tt.expectFile)
				if result != expectedPath {
					t.Errorf("findClosestJournalFile() = %v, want %v", result, expectedPath)
				}
			}
		})
	}
}

func TestCmdNew(t *testing.T) {
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	config := &Config{RootDir: tempDir}

	// Create a previous journal to use as source
	prevJournal := filepath.Join(tempDir, "2024/01/2024-01-01.md")
	createTestFile(t, prevJournal, `---
date: 2024-01-01
---

# Daily Journal

## Todos

- [ ] Previous task
- [x] Completed task

## Notes

Previous notes.
`)

	tests := []struct {
		name        string
		rootDir     string
		expectError bool
	}{
		{
			name:        "successful creation",
			rootDir:     tempDir,
			expectError: false,
		},
		{
			name:        "invalid root directory",
			rootDir:     "/nonexistent/path",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmdNew(tt.rootDir, "", config)

			if tt.expectError {
				if err == nil {
					t.Errorf("cmdNew() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("cmdNew() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCmdNew_AlreadyExists(t *testing.T) {
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	config := &Config{RootDir: tempDir}

	// Create journal for today
	today := time.Now().Format("2006-01-02")
	year := time.Now().Format("2006")
	month := time.Now().Format("01")
	expectedPath := filepath.Join(tempDir, year, month, today+".md")
	createTestFile(t, expectedPath, "existing content")

	// Should not error if file already exists
	err := cmdNew(tempDir, "", config)
	if err != nil {
		t.Errorf("cmdNew() unexpected error when file exists: %v", err)
	}
}

func TestValidateFilePath(t *testing.T) {
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

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
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

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
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

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
				Custom: map[string]interface{}{
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
				Custom: map[string]interface{}{
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
				Custom: map[string]interface{}{
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
				Custom: map[string]interface{}{
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
		custom      map[string]interface{}
		expectError bool
	}{
		{
			name:        "nil custom variables",
			custom:      nil,
			expectError: false,
		},
		{
			name:        "empty custom variables",
			custom:      map[string]interface{}{},
			expectError: false,
		},
		{
			name: "valid custom variables",
			custom: map[string]interface{}{
				"author":    "John Doe",
				"project":   "Test",
				"version":   1,
				"active":    true,
				"tags":      []string{"test", "demo"},
				"numbers":   []int{1, 2, 3},
				"mixed":     []interface{}{"string", 42, true},
				"_private":  "value",
				"CamelCase": "value",
			},
			expectError: false,
		},
		{
			name: "reserved variable name",
			custom: map[string]interface{}{
				"TODOS": "invalid",
			},
			expectError: true,
		},
		{
			name: "invalid variable name - starts with number",
			custom: map[string]interface{}{
				"123invalid": "value",
			},
			expectError: true,
		},
		{
			name: "invalid variable name - contains special chars",
			custom: map[string]interface{}{
				"test-value": "value",
			},
			expectError: true,
		},
		{
			name: "unsupported type - complex number",
			custom: map[string]interface{}{
				"complex": complex(1, 2),
			},
			expectError: true,
		},
		{
			name: "unsupported type - map",
			custom: map[string]interface{}{
				"nested": map[string]string{"key": "value"},
			},
			expectError: true,
		},
		{
			name: "unsupported type in array",
			custom: map[string]interface{}{
				"mixed": []interface{}{"valid", complex(1, 2)},
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

func TestIsValidVariableName(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		expected bool
	}{
		{"empty string", "", false},
		{"valid simple name", "test", true},
		{"valid with underscore", "test_var", true},
		{"valid starting with underscore", "_private", true},
		{"valid with numbers", "var123", true},
		{"invalid starting with number", "123var", false},
		{"invalid with dash", "test-var", false},
		{"invalid with space", "test var", false},
		{"invalid with special chars", "test@var", false},
		{"valid camelCase", "camelCase", true},
		{"valid PascalCase", "PascalCase", true},
		{"valid UPPERCASE", "UPPERCASE", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidVariableName(tt.varName)
			if result != tt.expected {
				t.Errorf("isValidVariableName(%q) = %v, expected %v", tt.varName, result, tt.expected)
			}
		})
	}
}

func TestIsValidVariableType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"string", "test", true},
		{"int", 42, true},
		{"int32", int32(42), true},
		{"int64", int64(42), true},
		{"float32", float32(3.14), true},
		{"float64", 3.14, true},
		{"bool", true, true},
		{"string slice", []string{"a", "b"}, true},
		{"int slice", []int{1, 2, 3}, true},
		{"interface slice with valid types", []interface{}{"string", 42, true}, true},
		{"interface slice with invalid type", []interface{}{"string", complex(1, 2)}, false},
		{"complex number", complex(1, 2), false},
		{"map", map[string]string{"key": "value"}, false},
		{"struct", struct{ Name string }{Name: "test"}, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidVariableType(tt.value)
			if result != tt.expected {
				t.Errorf("isValidVariableType(%v) = %v, expected %v", tt.value, result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkExpandPath(b *testing.B) {
	paths := []string{
		"/absolute/path",
		"relative/path",
		"~/home/path",
		"",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			expandPath(path)
		}
	}
}

func BenchmarkResolveTemplate(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "todoer-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	templateFile := filepath.Join(tempDir, "template.md")
	if err := os.WriteFile(templateFile, []byte("Test template content"), 0644); err != nil {
		b.Fatalf("Failed to create template file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolveTemplate(templateFile)
	}
}
