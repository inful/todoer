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
			err := processJournal(tt.sourceFile, tt.targetFile, "", tt.templateDate, false, false, config, logger)

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
	err := processJournal(sourceFile, targetFile, "", "", false, false, config, logger)
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

	if err := processJournal(sourceFile, targetFile, "", "", true, false, config, logger); err != nil {
		t.Fatalf("processJournal() unexpected error: %v", err)
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
			err := cmdNew(tt.rootDir, "", false, config, logger)

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
	err := cmdNew(tempDir, "", false, config, logger)
	if err != nil {
		t.Errorf("cmdNew() unexpected error when file exists: %v", err)
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
	if err := cmdNew(tempDir, "", false, config, logger); err != nil {
		t.Fatalf("cmdNew() unexpected error: %v", err)
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
