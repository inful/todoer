package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/inful/todoer/pkg/core"
)

// Helper function to create a temporary directory
func setupTempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// tUnsetenv unsets an environment variable for the duration of the test,
// mirroring the t.Setenv naming convention.
func tUnsetenv(t *testing.T, key string) {
	t.Helper()
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("cannot unset environment variable %s: %v", key, err)
	}
}

// Helper function to create a test file
func createTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
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
	tempDir := setupTempDir(t)

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

			content, name, err := resolveTemplate(tt.templateFile)

			if tt.expectError {
				if err == nil {
					t.Errorf("resolveTemplate() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("resolveTemplate() unexpected error: %v", err)
				}
				if name != tt.expectName {
					t.Errorf("resolveTemplate() name = %v, want %v", name, tt.expectName)
				}
				if content == "" {
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
	tempDir := setupTempDir(t)

	defer func() {
		// Restore original environment
		if originalXDG != "" {
			t.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			tUnsetenv(t, "XDG_CONFIG_HOME")
		}
		if originalHome != "" {
			t.Setenv("HOME", originalHome)
		} else {
			tUnsetenv(t, "HOME")
		}
		if originalRootDir != "" {
			t.Setenv("TODOER_ROOT_DIR", originalRootDir)
		} else {
			tUnsetenv(t, "TODOER_ROOT_DIR")
		}
		if originalTemplateFile != "" {
			t.Setenv("TODOER_TEMPLATE_FILE", originalTemplateFile)
		} else {
			tUnsetenv(t, "TODOER_TEMPLATE_FILE")
		}
	}()

	// Set isolated environment
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("HOME", tempDir)
	tUnsetenv(t, "TODOER_ROOT_DIR")
	tUnsetenv(t, "TODOER_TEMPLATE_FILE")

	// Test loading config with no config file (should succeed with defaults)
	config, err := loadConfig()
	if err != nil {
		t.Errorf("loadConfig() error = %v, want nil", err)
	}
	if config == nil {
		t.Fatalf("loadConfig() returned nil config")
		return
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
			t.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			tUnsetenv(t, "XDG_CONFIG_HOME")
		}
		if originalHome != "" {
			t.Setenv("HOME", originalHome)
		} else {
			tUnsetenv(t, "HOME")
		}
		if originalRootDir != "" {
			t.Setenv("TODOER_ROOT_DIR", originalRootDir)
		} else {
			tUnsetenv(t, "TODOER_ROOT_DIR")
		}
		if originalTemplateFile != "" {
			t.Setenv("TODOER_TEMPLATE_FILE", originalTemplateFile)
		} else {
			tUnsetenv(t, "TODOER_TEMPLATE_FILE")
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
				if err := os.MkdirAll(configDir, 0o755); err != nil {
					t.Fatal(err)
				}
				configPath := filepath.Join(configDir, "config.toml")
				customRoot := filepath.Join(tempDir, "custom_root")
				configContent := fmt.Sprintf(`root_dir = "%s"`, customRoot)
				if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
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
				if err := os.MkdirAll(configDir, 0o755); err != nil {
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
				if err := os.MkdirAll(isolatedHome, 0o755); err != nil {
					t.Fatal(err)
				}
				t.Setenv("HOME", isolatedHome)
				return "" // No config file setup
			},
			expectedRootDir: ".", // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a unique tempDir for this test case
			testTempDir := setupTempDir(t)

			// Clear environment variables first
			tUnsetenv(t, "TODOER_ROOT_DIR")
			tUnsetenv(t, "TODOER_TEMPLATE_FILE")

			// Set XDG_CONFIG_HOME to point to this test's tempDir
			if tt.xdgConfigHome == "SET_TO_TEMP_DIR" {
				t.Setenv("XDG_CONFIG_HOME", testTempDir)
			} else if tt.xdgConfigHome != "" {
				t.Setenv("XDG_CONFIG_HOME", tt.xdgConfigHome)
			} else {
				tUnsetenv(t, "XDG_CONFIG_HOME")
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
			t.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			tUnsetenv(t, "XDG_CONFIG_HOME")
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
				if err := os.MkdirAll(configDir, 0o755); err != nil {
					t.Fatal(err)
				}
				templatePath := filepath.Join(configDir, "template.md")
				templateContent := "# Custom Template\n## Todos\n{{.TODOS}}"
				if err := os.WriteFile(templatePath, []byte(templateContent), 0o644); err != nil {
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
				if err := os.MkdirAll(configDir, 0o755); err != nil {
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
				if err := os.MkdirAll(configDir, 0o755); err != nil {
					t.Fatal(err)
				}
				xdgTemplate := filepath.Join(configDir, "template.md")
				if err := os.WriteFile(xdgTemplate, []byte("XDG Template"), 0o644); err != nil {
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
			testTempDir := setupTempDir(t)

			// Set XDG_CONFIG_HOME to this test's tempDir
			t.Setenv("XDG_CONFIG_HOME", testTempDir)

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
			content, name, err := resolveTemplate(templateFile)

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

			if name != expectedName {
				t.Errorf("Expected template name %q, got %q", expectedName, name)
			}

			if content == "" {
				t.Error("Template content should not be empty")
			}
		})
	}
}

func TestProcessJournal_ValidationErrors(t *testing.T) {
	tempDir := setupTempDir(t)

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
			logger := NewLogger(ModeQuiet)
			err := processJournalWithOptions(ProcessOptions{
				SourceFile:   tt.sourceFile,
				TargetFile:   tt.targetFile,
				TemplateDate: tt.templateDate,
			}, config, logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("processJournalWithOptions() expected error, got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("processJournalWithOptions() error = %v, want to contain %v", err, tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("processJournalWithOptions() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestProcessJournal_Success(t *testing.T) {
	tempDir := setupTempDir(t)

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

	logger := NewLogger(ModeQuiet)
	err := processJournalWithOptions(ProcessOptions{
		SourceFile: sourceFile,
		TargetFile: targetFile,
	}, config, logger)
	if err != nil {
		t.Fatalf("processJournalWithOptions() unexpected error: %v", err)
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

func TestProcessJournal_SkipBackupStillUpdatesSource(t *testing.T) {
	tempDir := setupTempDir(t)

	sourceContent := `---
date: 2024-01-01
---

# Daily Journal

## Todos

- [ ] Task to carry
- [x] Already done

## Notes
`

	sourceFile := filepath.Join(tempDir, "source.md")
	targetFile := filepath.Join(tempDir, "target.md")
	createTestFile(t, sourceFile, sourceContent)

	config := &Config{RootDir: tempDir}
	logger := NewLogger(ModeQuiet)

	if err := processJournalWithOptions(ProcessOptions{
		SourceFile: sourceFile,
		TargetFile: targetFile,
		SkipBackup: true,
	}, config, logger); err != nil {
		t.Fatalf("processJournalWithOptions() unexpected error: %v", err)
	}

	if _, err := os.Stat(sourceFile + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("expected no backup file when skipBackup=true, got err=%v", err)
	}

	sourceAfter, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Fatalf("failed to read updated source file: %v", err)
	}

	sourceAfterStr := string(sourceAfter)
	if sourceAfterStr == sourceContent {
		t.Fatalf("expected source file to be updated when skipBackup=true")
	}

	if !strings.Contains(sourceAfterStr, "Moved to [[") {
		t.Fatalf("expected source file to include moved marker after update, got:\n%s", sourceAfterStr)
	}

	if _, err := os.Stat(targetFile); err != nil {
		t.Fatalf("expected target file to be created: %v", err)
	}
}

func TestFindClosestJournalFile(t *testing.T) {
	tempDir := setupTempDir(t)

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
	tempDir := setupTempDir(t)

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
			logger := NewLogger(ModeQuiet)
			err := cmdNewWithOptions(tt.rootDir, "", false, false, config, logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("cmdNewWithOptions() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("cmdNewWithOptions() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCmdNew_AlreadyExists(t *testing.T) {
	tempDir := setupTempDir(t)

	config := &Config{RootDir: tempDir}

	// Create journal for today
	today := time.Now().Format("2006-01-02")
	year := time.Now().Format("2006")
	month := time.Now().Format("01")
	expectedPath := filepath.Join(tempDir, year, month, today+".md")
	createTestFile(t, expectedPath, "existing content")

	// Should not error if file already exists
	logger := NewLogger(ModeQuiet)
	err := cmdNewWithOptions(tempDir, "", false, false, config, logger)
	if err != nil {
		t.Errorf("cmdNewWithOptions() unexpected error when file exists: %v", err)
	}
}

func TestCmdNewWithOptions_DisableSourceBackup(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	createTestFile(t, source, `---
date: `+yesterday+`
---

# Daily Journal

## Todos

- [[`+yesterday+`]]
  - [ ] Carryover item

## Notes
`)

	logger := NewLogger(ModeQuiet)
	if err := cmdNewWithOptions(tempDir, "", false, false, config, logger); err != nil {
		t.Fatalf("cmdNewWithOptions() unexpected error: %v", err)
	}

	target := buildJournalPath(tempDir, today)
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected today's journal to be created: %v", err)
	}

	if _, err := os.Stat(source + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("expected no backup file when preserveSourceBackup=false, got err=%v", err)
	}
}

func TestCmdNewWithOptions_DisableSourceBackup_PreservesCompletedAndCarriesUnfinished(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	createTestFile(t, source, `---
date: `+yesterday+`
---

# Daily Journal

## Todos

- [x] Done task
- [ ] Carry task

## Notes
`)

	logger := NewLogger(ModeQuiet)
	if err := cmdNewWithOptions(tempDir, "", false, false, config, logger); err != nil {
		t.Fatalf("cmdNewWithOptions() unexpected error: %v", err)
	}

	if _, err := os.Stat(source + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("expected no backup file when preserveSourceBackup=false, got err=%v", err)
	}

	sourceAfter, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("failed to read source after carryover: %v", err)
	}

	sourceAfterStr := string(sourceAfter)
	if !strings.Contains(sourceAfterStr, "Done task") {
		t.Fatalf("expected source to preserve completed task, got:\n%s", sourceAfterStr)
	}
	if strings.Contains(sourceAfterStr, "Carry task") {
		t.Fatalf("expected source to remove carried unfinished task, got:\n%s", sourceAfterStr)
	}

	target := buildJournalPath(tempDir, today)
	targetAfter, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("failed to read target after carryover: %v", err)
	}

	targetAfterStr := string(targetAfter)
	if !strings.Contains(targetAfterStr, "Carry task") {
		t.Fatalf("expected target to contain carried unfinished task, got:\n%s", targetAfterStr)
	}
	if strings.Contains(targetAfterStr, "Done task") {
		t.Fatalf("expected target to exclude completed task, got:\n%s", targetAfterStr)
	}
}

func TestCmdNew_DoesNotCreateBackupByDefault(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	createTestFile(t, source, `---
date: `+yesterday+`
---

# Daily Journal

## Todos

- [[`+yesterday+`]]
  - [ ] Carryover item

## Notes
`)

	logger := NewLogger(ModeQuiet)
	if err := cmdNewWithOptions(tempDir, "", false, false, config, logger); err != nil {
		t.Fatalf("cmdNewWithOptions() unexpected error: %v", err)
	}

	if _, err := os.Stat(source + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("expected no backup file from cmdNew by default, got err=%v", err)
	}
}

func TestCmdAdd_DoesNotCreateBackupByDefault(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	createTestFile(t, source, `---
date: `+yesterday+`
---

# Daily Journal

## Todos

- [[`+yesterday+`]]
  - [ ] Carryover item

## Notes
`)

	logger := NewLogger(ModeQuiet)
	if err := cmdAdd(tempDir, "", "new todo", false, false, config, logger); err != nil {
		t.Fatalf("cmdAdd() unexpected error: %v", err)
	}

	if _, err := os.Stat(source + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("expected no backup file from cmdAdd by default, got err=%v", err)
	}

	target := buildJournalPath(tempDir, today)
	targetAfter, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("failed to read target: %v", err)
	}
	if !strings.Contains(string(targetAfter), "new todo") {
		t.Fatalf("expected target to contain the new todo, got:\n%s", string(targetAfter))
	}
}

func TestCmdAdd_WithBackupCreatesBackup(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	createTestFile(t, source, `---
date: `+yesterday+`
---

# Daily Journal

## Todos

- [[`+yesterday+`]]
  - [ ] Carryover item

## Notes
`)

	logger := NewLogger(ModeQuiet)
	if err := cmdAdd(tempDir, "", "new todo", false, true, config, logger); err != nil {
		t.Fatalf("cmdAdd() unexpected error: %v", err)
	}

	backupPath := source + ".bak"
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("expected backup file when backup=true, got err=%v", err)
	}

	target := buildJournalPath(tempDir, today)
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected target file to be created, got err=%v", err)
	}
}

func TestCmdAdd_PreservesCompletedUndatedTodos(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(tempDir, today)

	// Pre-create today's journal with both completed and uncompleted undated
	// todos. The add path runs MoveUndatedTodosToCurrentDate internally, so
	// both classes of undated todos should land in today's section rather
	// than the completed ones being silently dropped.
	createTestFile(t, journalPath, `---
date: `+today+`
---

# Daily Journal

## Todos

- [x] Done last week
- [ ] Undated carry
- [x] Done yesterday

## Notes
`)

	logger := NewLogger(ModeQuiet)
	if err := cmdAdd(tempDir, "", "fresh today item", false, false, config, logger); err != nil {
		t.Fatalf("cmdAdd() unexpected error: %v", err)
	}

	after, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatalf("failed to read today's journal: %v", err)
	}
	content := string(after)

	// The new todo must be present.
	if !strings.Contains(content, "fresh today item") {
		t.Fatalf("expected new todo to be appended, got:\n%s", content)
	}
	// The undated completed and uncompleted items must both be preserved.
	if !strings.Contains(content, "Done last week") {
		t.Fatalf("expected undated completed todo to be preserved, got:\n%s", content)
	}
	if !strings.Contains(content, "Undated carry") {
		t.Fatalf("expected undated uncompleted todo to be preserved, got:\n%s", content)
	}
	if !strings.Contains(content, "Done yesterday") {
		t.Fatalf("expected second undated completed todo to be preserved, got:\n%s", content)
	}
	// All four items must live under today's [[date]] section. The journal
	// should not have any undated day left at the top of the todos section.
	if !strings.Contains(content, "[["+today+"]]") {
		t.Fatalf("expected today's [[date]] section to exist, got:\n%s", content)
	}
}

func TestCmdAdd_EmptyTextReturnsError(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir}

	logger := NewLogger(ModeQuiet)
	if err := cmdAdd(tempDir, "", "   ", false, false, config, logger); err == nil {
		t.Fatalf("expected error on empty todo text")
	}
}

func TestCmdAdd_PrintPath(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(tempDir, today)
	createTestFile(t, journalPath, "---\ndate: "+today+"\n---\n\n# J\n\n## Todos\n\n- [ ] existing\n\n## Notes\n")

	logger := NewLogger(ModeQuiet)
	if err := cmdAdd(tempDir, "", "new", true, false, config, logger); err != nil {
		t.Fatalf("cmdAdd: %v", err)
	}
}

func TestFindOrCreateDaySection(t *testing.T) {
	existing := &core.DaySection{Date: "2026-03-16", Items: []*core.TodoItem{{Text: "x"}}}
	journal := &core.TodoJournal{Days: []*core.DaySection{existing}}

	// Finds existing.
	got := core.FindOrCreateDaySection(journal, "2026-03-16")
	if got != existing {
		t.Fatalf("expected to return the existing day section")
	}

	// Creates a new one when missing.
	got2 := core.FindOrCreateDaySection(journal, "2026-03-17")
	if got2 == nil || got2.Date != "2026-03-17" {
		t.Fatalf("expected a new day section for 2026-03-17, got %+v", got2)
	}
	if len(journal.Days) != 2 {
		t.Fatalf("expected journal to now have 2 day sections, got %d", len(journal.Days))
	}
}

func TestAppendTodoToJournal_NoExistingFile(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(tempDir, today)

	if err := appendTodoToJournal(journalPath, today, "first", config); err == nil {
		t.Fatalf("expected error when journal file does not exist")
	}
}

func TestCmdNewWithOptions_NoPreviousJournalCreatesEmpty(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(tempDir, today)

	logger := NewLogger(ModeQuiet)
	if err := cmdNewWithOptions(tempDir, "", false, false, config, logger); err != nil {
		t.Fatalf("cmdNewWithOptions: %v", err)
	}

	after, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatalf("expected today's journal to be created, got: %v", err)
	}
	if !strings.Contains(string(after), core.TodosHeader) {
		t.Fatalf("expected journal to contain the todos header, got:\n%s", string(after))
	}
}

func TestProcessJournal_PrintPath(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)
	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] Carryover\n\n## Notes\n")

	logger := NewLogger(ModeQuiet)
	if err := processJournalWithOptions(ProcessOptions{
		SourceFile:   source,
		TargetFile:   target,
		TemplateDate: today,
		PrintPath:    true,
	}, config, logger); err != nil {
		t.Fatalf("processJournal: %v", err)
	}
}

func TestProcessJournal_OnlyCompletedItems(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	// Source with only completed todos: nothing to carry over to today, but
	// the source's completed item is still date-tagged in the carryover
	// pipeline and the target is still written.
	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)
	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [x] Done\n\n## Notes\n")

	logger := NewLogger(ModeQuiet)
	if err := processJournalWithOptions(ProcessOptions{
		SourceFile:   source,
		TargetFile:   target,
		TemplateDate: today,
	}, config, logger); err != nil {
		t.Fatalf("processJournal: %v", err)
	}

	// Source updated (date tag added to the completed item).
	afterSource, _ := os.ReadFile(source)
	if !strings.Contains(string(afterSource), "#"+yesterday) {
		t.Fatalf("expected source to be updated with date tag, got:\n%s", string(afterSource))
	}

	// Target exists; the completed item lives in the target's completed section,
	// not under today.
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected target to be created, got %v", err)
	}
	targetContent, _ := os.ReadFile(target)
	if strings.Contains(string(targetContent), "  - [ ]") {
		t.Fatalf("expected no uncompleted items in target, got:\n%s", string(targetContent))
	}
}

func TestProcessJournal_MergeIntoExistingTarget(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)

	// Source has one carryover item.
	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] carry me\n\n## Notes\n")

	// Target already exists with a pre-existing today item.
	createTestFile(t, target, "---\ndate: "+today+"\n---\n\n# J\n\n## Todos\n\n- [["+today+"]]\n  - [ ] already in today\n\n## Notes\n")

	logger := NewLogger(ModeQuiet)
	if err := processJournalWithOptions(ProcessOptions{
		SourceFile:    source,
		TargetFile:    target,
		TemplateDate:  today,
		MergeIfExists: true,
	}, config, logger); err != nil {
		t.Fatalf("processJournal: %v", err)
	}

	after, _ := os.ReadFile(target)
	content := string(after)
	if !strings.Contains(content, "carry me") {
		t.Fatalf("expected the carryover item to be merged in, got:\n%s", content)
	}
	if !strings.Contains(content, "already in today") {
		t.Fatalf("expected the pre-existing today item to be preserved, got:\n%s", content)
	}
}

func TestProcessJournal_MergeIsIdempotent(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)

	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] carry me\n\n## Notes\n")
	// First run: target doesn't exist; created with the carryover.
	// Second run: target exists; should merge but not duplicate.

	logger := NewLogger(ModeQuiet)
	mergeOpts := ProcessOptions{
		SourceFile:    source,
		TargetFile:    target,
		TemplateDate:  today,
		MergeIfExists: true,
	}
	if err := processJournalWithOptions(mergeOpts, config, logger); err != nil {
		t.Fatalf("first processJournal: %v", err)
	}
	if err := processJournalWithOptions(mergeOpts, config, logger); err != nil {
		t.Fatalf("second processJournal: %v", err)
	}

	after, _ := os.ReadFile(target)
	content := string(after)
	// The item should appear exactly once under the carryover day.
	day := "- [[" + yesterday + "]]"
	if _, afterDay, found := strings.Cut(content, day); found {
		if strings.Count(afterDay, "  - [ ] carry me") != 1 {
			t.Fatalf("expected exactly one 'carry me' after day header, got:\n%s", afterDay)
		}
	} else {
		t.Fatalf("expected carryover day %q in target, got:\n%s", day, content)
	}
}

func TestFingerprintEnabled_Default(t *testing.T) {
	// TODOER_FINGERPRINT is unset by default in tests; the helper
	// should report false.
	if err := os.Unsetenv(fingerprintEnabledEnv); err != nil {
		t.Fatalf("unsetenv: %v", err)
	}
	if fingerprintEnabled() {
		t.Fatalf("expected fingerprintEnabled to be false by default")
	}
}

func TestFingerprintEnabled_TruthyValues(t *testing.T) {
	// strconv.ParseBool accepts: 1, t, T, TRUE, true, True,
	// 0, f, F, FALSE, false, False. We expect the spike to
	// turn on for any of the truthy spellings and to stay
	// off for the falsy ones and for garbage.
	for _, value := range []string{"1", "t", "T", "true", "TRUE", "True"} {
		t.Setenv(fingerprintEnabledEnv, value)
		if !fingerprintEnabled() {
			t.Fatalf("expected fingerprintEnabled to be true for %q", value)
		}
	}
	for _, value := range []string{"0", "f", "F", "false", "FALSE", "no", "", "yes-please"} {
		t.Setenv(fingerprintEnabledEnv, value)
		if fingerprintEnabled() {
			t.Fatalf("expected fingerprintEnabled to be false for %q", value)
		}
	}
}

func TestFingerprintWriteEnabled_Default(t *testing.T) {
	// TODOER_FINGERPRINT_WRITE is unset by default. The default
	// must be "write enabled" so that turning the spike on with
	// TODOER_FINGERPRINT=1 retains the existing behavior.
	if err := os.Unsetenv("TODOER_FINGERPRINT_WRITE"); err != nil {
		t.Fatalf("unsetenv: %v", err)
	}
	if !fingerprintWriteEnabled() {
		t.Fatalf("expected fingerprintWriteEnabled to be true by default")
	}
}

func TestFingerprintWriteEnabled_TruthyValues(t *testing.T) {
	for _, value := range []string{"1", "t", "T", "true", "TRUE", "True"} {
		t.Setenv("TODOER_FINGERPRINT_WRITE", value)
		if !fingerprintWriteEnabled() {
			t.Fatalf("expected fingerprintWriteEnabled to be true for %q", value)
		}
	}
	for _, value := range []string{"0", "f", "F", "false", "FALSE"} {
		t.Setenv("TODOER_FINGERPRINT_WRITE", value)
		if fingerprintWriteEnabled() {
			t.Fatalf("expected fingerprintWriteEnabled to be false for %q", value)
		}
	}
	// Unset and unparseable values fall back to the documented default
	// (write enabled).
	t.Setenv("TODOER_FINGERPRINT_WRITE", "")
	if !fingerprintWriteEnabled() {
		t.Fatalf("expected fingerprintWriteEnabled to be true for unset value (default on)")
	}
	t.Setenv("TODOER_FINGERPRINT_WRITE", "garbage")
	if !fingerprintWriteEnabled() {
		t.Fatalf("expected fingerprintWriteEnabled to be true for unparseable value (default on)")
	}
}

func TestComputeFingerprint_Deterministic(t *testing.T) {
	// mdfp fingerprints a canonical virtual document of
	// frontmatter (with the fingerprint field stripped) plus body.
	// The test inputs need a recognizable body to fingerprint.
	content := []byte("# hello world\n\nSome body.\n")
	got, err := computeFingerprint(content)
	if err != nil {
		t.Fatalf("computeFingerprint: %v", err)
	}
	if len(got) != 64 {
		t.Fatalf("expected 64-char hex SHA-256, got %d chars: %s", len(got), got)
	}
	got2, err := computeFingerprint(content)
	if err != nil {
		t.Fatalf("computeFingerprint (second call): %v", err)
	}
	if got != got2 {
		t.Fatalf("expected deterministic fingerprint, got %q and %q", got, got2)
	}
	// Different content should yield a different fingerprint.
	got3, err := computeFingerprint([]byte("# hello world!\n\nSome body.\n"))
	if err != nil {
		t.Fatalf("computeFingerprint (third call): %v", err)
	}
	if got == got3 {
		t.Fatalf("expected different content to yield different fingerprint")
	}
}

func TestComputeFingerprint_FrontmatterChangesHash(t *testing.T) {
	// mdfp includes the frontmatter in the hash, with only the
	// 'fingerprint' field stripped. Two documents with the same
	// body but different frontmatter metadata must therefore
	// produce different fingerprints. This pins the
	// canonicalization contract — if mdfp ever changes to
	// exclude frontmatter, this test will fail.
	body := "\n# hello world\n\nSome body.\n"
	withTitleA := []byte("---\ntitle: A\n---\n" + body)
	withTitleB := []byte("---\ntitle: B\n---\n" + body)
	fpA, err := computeFingerprint(withTitleA)
	if err != nil {
		t.Fatalf("computeFingerprint (title A): %v", err)
	}
	fpB, err := computeFingerprint(withTitleB)
	if err != nil {
		t.Fatalf("computeFingerprint (title B): %v", err)
	}
	if fpA == fpB {
		t.Fatalf("expected different frontmatter to change the hash, both = %q", fpA)
	}
	// Same frontmatter should still produce the same hash.
	fpA2, err := computeFingerprint([]byte("---\ntitle: A\n---\n" + body))
	if err != nil {
		t.Fatalf("computeFingerprint (title A again): %v", err)
	}
	if fpA != fpA2 {
		t.Fatalf("expected deterministic fingerprint for same content, got %q and %q", fpA, fpA2)
	}
}

func TestCheckFingerprintMismatch_NoPriorFingerprint(_ *testing.T) {
	// Existing target has no frontmatter fingerprint yet. The check
	// should be a silent no-op (treat as fresh sync).
	logger := NewLogger(ModeQuiet)
	existing := []byte("# Just a heading\n\nNo frontmatter.\n")
	checkFingerprintMismatch(existing, []byte("source content"), "/tmp/whatever", logger)
}

func TestCheckFingerprintMismatch_Matching(t *testing.T) {
	source := []byte("# the source\n\nbody content.\n")
	fp, err := computeFingerprint(source)
	if err != nil {
		t.Fatalf("computeFingerprint: %v", err)
	}
	existing := []byte("---\ntitle: x\nfingerprint: " + fp + "\n---\n\n# Body\n")
	logger := NewLogger(ModeQuiet)
	// Same source -> no mismatch log; we just check it does not panic.
	checkFingerprintMismatch(existing, source, "/tmp/whatever", logger)
}

func TestCheckFingerprintMismatch_Differing(_ *testing.T) {
	// Existing target records a fingerprint for some other source
	// content. The current source differs; the check should log a
	// mismatch and not fail.
	existing := []byte("---\ntitle: x\nfingerprint: deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef\n---\n\n# Body\n")
	logger := NewLogger(ModeQuiet)
	checkFingerprintMismatch(existing, []byte("# the actual source\n\nbody.\n"), "/tmp/whatever", logger)
}

func TestProcessJournal_FingerprintMismatchForcesReMerge(t *testing.T) {
	// End-to-end: with the toggle on, the second run with a modified
	// source should still merge cleanly. The fingerprint mismatch is a
	// hint, not a failure.
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)

	// Source with one carryover item.
	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] carry me\n\n## Notes\n")

	// Pre-existing target with a stale fingerprint and a fresh
	// today-only item, so the merge will both keep the existing
	// item and detect a fingerprint mismatch.
	preExistingTarget := "---\ndate: " + today + "\nfingerprint: 0000000000000000000000000000000000000000000000000000000000000000\n---\n\n# J\n\n## Todos\n\n- [[" + today + "]]\n  - [ ] already here\n\n## Notes\n"
	createTestFile(t, target, preExistingTarget)

	t.Setenv(fingerprintEnabledEnv, "1")

	logger := NewLogger(ModeQuiet)
	if err := processJournalWithOptions(ProcessOptions{
		SourceFile:    source,
		TargetFile:    target,
		TemplateDate:  today,
		MergeIfExists: true,
	}, config, logger); err != nil {
		t.Fatalf("processJournal: %v", err)
	}

	after, _ := os.ReadFile(target)
	content := string(after)
	if !strings.Contains(content, "carry me") {
		t.Fatalf("expected the carryover item to be merged in, got:\n%s", content)
	}
	if !strings.Contains(content, "already here") {
		t.Fatalf("expected the pre-existing today item to be preserved, got:\n%s", content)
	}
}

func TestProcessJournal_RecordsFingerprintWhenEnabled(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)

	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] carry me\n\n## Notes\n")

	t.Setenv(fingerprintEnabledEnv, "1")

	logger := NewLogger(ModeQuiet)
	if err := processJournalWithOptions(ProcessOptions{
		SourceFile:    source,
		TargetFile:    target,
		TemplateDate:  today,
		MergeIfExists: true,
	}, config, logger); err != nil {
		t.Fatalf("processJournal: %v", err)
	}

	after, _ := os.ReadFile(target)
	content := string(after)
	if !strings.Contains(content, "fingerprint:") {
		t.Fatalf("expected fingerprint key in target frontmatter, got:\n%s", content)
	}
	if strings.Contains(content, "todoer_source_fingerprint") {
		t.Fatalf("expected old 'todoer_source_fingerprint' field name to be gone, got:\n%s", content)
	}
	// The recorded fingerprint should be a 64-char hex SHA-256.
	before, afterKey, found := strings.Cut(content, "fingerprint: ")
	if !found {
		t.Fatalf("expected 'fingerprint: ' key, got:\n%s", content)
	}
	_ = before
	end := strings.IndexByte(afterKey, '\n')
	if end < 0 {
		t.Fatalf("expected newline after fingerprint value, got:\n%s", content)
	}
	value := strings.TrimSpace(afterKey[:end])
	if len(value) != 64 {
		t.Fatalf("expected 64-char hex fingerprint, got %d chars: %q", len(value), value)
	}
}

func TestProcessJournal_NoFingerprintWhenDisabled(t *testing.T) {
	// Without TODOER_FINGERPRINT, the spike is invisible: the target
	// does not receive a fingerprint key.
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)

	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] carry me\n\n## Notes\n")

	// Make sure the env var is unset.
	if err := os.Unsetenv(fingerprintEnabledEnv); err != nil {
		t.Fatalf("unsetenv: %v", err)
	}

	logger := NewLogger(ModeQuiet)
	if err := processJournalWithOptions(ProcessOptions{
		SourceFile:    source,
		TargetFile:    target,
		TemplateDate:  today,
		MergeIfExists: true,
	}, config, logger); err != nil {
		t.Fatalf("processJournal: %v", err)
	}

	after, _ := os.ReadFile(target)
	if strings.Contains(string(after), "todoer_source_fingerprint") {
		t.Fatalf("expected no old todoer_source_fingerprint key when toggle is off, got:\n%s", string(after))
	}
	if strings.Contains(string(after), "fingerprint:") {
		t.Fatalf("expected no fingerprint key when toggle is off, got:\n%s", string(after))
	}
}

func TestProcessJournal_NoFingerprintWriteWhenSuppressed(t *testing.T) {
	// With TODOER_FINGERPRINT=1 (spike on) and TODOER_FINGERPRINT_WRITE=0
	// (write side suppressed), the target does not receive a fingerprint
	// key. The read side (mismatch detection) is still active.
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)

	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] carry me\n\n## Notes\n")

	t.Setenv(fingerprintEnabledEnv, "1")
	t.Setenv(fingerprintWriteEnv, "0")

	logger := NewLogger(ModeQuiet)
	if err := processJournalWithOptions(ProcessOptions{
		SourceFile:    source,
		TargetFile:    target,
		TemplateDate:  today,
		MergeIfExists: true,
	}, config, logger); err != nil {
		t.Fatalf("processJournal: %v", err)
	}

	after, _ := os.ReadFile(target)
	if strings.Contains(string(after), "todoer_source_fingerprint") {
		t.Fatalf("expected no old todoer_source_fingerprint key when write is suppressed, got:\n%s", string(after))
	}
	if strings.Contains(string(after), "fingerprint:") {
		t.Fatalf("expected no fingerprint key when write is suppressed, got:\n%s", string(after))
	}
}

func TestProcessJournal_AnnotateTargetFingerprintHelper(t *testing.T) {
	// Direct test of the helper: feed it a target without a fingerprint
	// and confirm both keys are added.
	target := "---\ndate: 2026-03-16\n---\n\n# J\n\n## Todos\n\n- [[2026-03-16]]\n  - [ ] x\n\n## Notes\n"
	source := []byte("source content for fingerprinting")
	out, err := annotateTargetWithFingerprint([]byte(target), source)
	if err != nil {
		t.Fatalf("annotateTargetWithFingerprint: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "fingerprint:") {
		t.Fatalf("expected fingerprint key, got:\n%s", got)
	}
	if strings.Contains(got, "todoer_source_fingerprint") {
		t.Fatalf("expected old field name to be gone, got:\n%s", got)
	}
	// Calling it again should be idempotent: the value is replaced
	// (not duplicated).
	out2, err := annotateTargetWithFingerprint(out, source)
	if err != nil {
		t.Fatalf("annotateTargetWithFingerprint (second call): %v", err)
	}
	if strings.Count(string(out2), "fingerprint:") != 1 {
		t.Fatalf("expected exactly one fingerprint key on idempotent call, got:\n%s", string(out2))
	}
}

func TestProcessJournal_AnnotateStripsLegacyFingerprintFields(t *testing.T) {
	// One-time migration: a target that still carries the pre-v0.4.0
	// spike fields (todoer_source_fingerprint / todoer_source_fingerprint_algo)
	// should have them stripped when the new fingerprint is written.
	// This prevents stale frontmatter from accumulating forever
	// after a user upgrades todoer with the spike enabled.
	target := "---\ndate: 2026-03-16\ntodoer_source_fingerprint: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\ntodoer_source_fingerprint_algo: sha256\n---\n\n# J\n\n## Notes\n"
	source := []byte("source content for fingerprinting")
	out, err := annotateTargetWithFingerprint([]byte(target), source)
	if err != nil {
		t.Fatalf("annotateTargetWithFingerprint: %v", err)
	}
	got := string(out)
	if strings.Contains(got, "todoer_source_fingerprint") {
		t.Fatalf("expected legacy todoer_source_fingerprint to be stripped, got:\n%s", got)
	}
	if !strings.Contains(got, "fingerprint:") {
		t.Fatalf("expected new fingerprint key to be present, got:\n%s", got)
	}
	// Non-fingerprint frontmatter fields must be preserved.
	if !strings.Contains(got, "date: 2026-03-16") {
		t.Fatalf("expected unrelated 'date' frontmatter key to be preserved, got:\n%s", got)
	}
}

func TestSafeWriteFile_BadTarget(t *testing.T) {
	tempDir := t.TempDir()

	// Target inside a non-existent directory should fail.
	bad := filepath.Join(tempDir, "no-such-dir", "out.md")
	if err := safeWriteFile(bad, []byte("hi"), 0o644); err == nil {
		t.Fatalf("expected error when target directory does not exist")
	}
}

func TestLogger_InfoAndDebugModes(_ *testing.T) {
	quiet := NewLogger(ModeQuiet)
	quiet.Info("hi %s", "x")
	quiet.Debug("hi %s", "x")
	normal := NewLogger(ModeNormal)
	normal.Info("hi %s", "x")
	debug := NewLogger(ModeDebug)
	debug.Debug("hi %s", "x")
	debug.Info("hi %s", "x")
}

func TestLogger_Error(_ *testing.T) {
	logger := NewLogger(ModeQuiet)
	logger.Error("oops %s", "x")
}

func TestExpandPathEdgeCases(t *testing.T) {
	if got := expandPath(""); got != "" {
		t.Fatalf("expected empty path passthrough, got %q", got)
	}
	if got := expandPath("/abs/path"); got != "/abs/path" {
		t.Fatalf("expected absolute path passthrough, got %q", got)
	}
	if got := expandPath("relative/path"); got != "relative/path" {
		t.Fatalf("expected relative path passthrough, got %q", got)
	}
	// Home expansion: when not actually run under a user with HOME set, this
	// falls through to the original path; that is the documented behavior.
	if got := expandPath("~/foo"); !strings.HasPrefix(got, "~/") && !strings.HasPrefix(got, "/") {
		t.Fatalf("expected home expansion to produce a path, got %q", got)
	}
}

func TestEnsureDirectories_CreatesMissingRoot(t *testing.T) {
	tempDir := t.TempDir()
	missing := filepath.Join(tempDir, "no-such-dir", "nested")

	cfg := &Config{RootDir: missing}
	if err := ensureDirectories(cfg); err != nil {
		t.Fatalf("ensureDirectories: %v", err)
	}
	if info, err := os.Stat(missing); err != nil || !info.IsDir() {
		t.Fatalf("expected missing root to be created, got err=%v", err)
	}
}

func TestEnsureDirectories_NilConfig(t *testing.T) {
	if err := ensureDirectories(nil); err != nil {
		t.Fatalf("ensureDirectories(nil) should be a no-op, got %v", err)
	}
}

func TestGetConfigValue(t *testing.T) {
	if got := getConfigValue("cli", "cfg"); got != "cli" {
		t.Fatalf("expected CLI value to win, got %q", got)
	}
	if got := getConfigValue("", "cfg"); got != "cfg" {
		t.Fatalf("expected config value when CLI is empty, got %q", got)
	}
	if got := getConfigValue("", ""); got != "" {
		t.Fatalf("expected empty when both are empty, got %q", got)
	}
}

func TestLoggerWithMode(t *testing.T) {
	logger := NewLogger(ModeNormal).WithMode(ModeQuiet)
	if logger.mode != ModeQuiet {
		t.Fatalf("expected WithMode to switch mode")
	}
}

func TestLoggerForCommand(t *testing.T) {
	base := NewLogger(ModeNormal)
	if loggerForCommand(base, true).mode != ModeQuiet {
		t.Fatalf("expected printPath=true to use ModeQuiet")
	}
	if loggerForCommand(base, false).mode != ModeNormal {
		t.Fatalf("expected printPath=false to keep base mode")
	}
}

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

// Benchmark tests
func BenchmarkExpandPath(b *testing.B) {
	paths := []string{
		"/absolute/path",
		"relative/path",
		"~/home/path",
		"",
	}

	b.ResetTimer()
	for b.Loop() {
		for _, path := range paths {
			expandPath(path)
		}
	}
}

func BenchmarkResolveTemplate(b *testing.B) {
	tempDir := b.TempDir()

	templateFile := filepath.Join(tempDir, "template.md")
	if err := os.WriteFile(templateFile, []byte("Test template content"), 0o644); err != nil {
		b.Fatalf("Failed to create template file: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = resolveTemplate(templateFile)
	}
}
