package main

import (
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

	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
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
		},
		{
			name: "valid config with template",
			config: &Config{
				RootDir:      tempDir,
				TemplateFile: filepath.Join(tempDir, "template.md"),
			},
			expectError: false,
		},
		{
			name: "config with invalid root dir path",
			config: &Config{
				RootDir: "../../../etc",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.expectError {
				t.Errorf("validateConfig() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestSafeWriteFile(t *testing.T) {
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	tests := []struct {
		name        string
		filename    string
		content     []byte
		perm        os.FileMode
		expectError bool
	}{
		{
			name:     "successful write",
			filename: filepath.Join(tempDir, "test.txt"),
			content:  []byte("test content"),
			perm:     0644,
		},
		{
			name:     "write to subdirectory",
			filename: filepath.Join(tempDir, "subdir", "test.txt"),
			content:  []byte("test content"),
			perm:     0644,
		},
		{
			name:        "invalid directory",
			filename:    "/nonexistent/directory/test.txt",
			content:     []byte("test content"),
			perm:        0644,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create parent directory if needed for valid test cases
			if !tt.expectError && filepath.Dir(tt.filename) != tempDir {
				if err := os.MkdirAll(filepath.Dir(tt.filename), 0755); err != nil {
					t.Fatalf("Failed to create parent directory: %v", err)
				}
			}

			err := safeWriteFile(tt.filename, tt.content, tt.perm)

			if tt.expectError {
				if err == nil {
					t.Errorf("safeWriteFile() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("safeWriteFile() unexpected error: %v", err)
				}

				// Verify file was created with correct content
				content, err := os.ReadFile(tt.filename)
				if err != nil {
					t.Errorf("Failed to read written file: %v", err)
				}
				if string(content) != string(tt.content) {
					t.Errorf("File content mismatch: got %s, want %s", content, tt.content)
				}

				// Verify file permissions
				info, err := os.Stat(tt.filename)
				if err != nil {
					t.Errorf("Failed to stat file: %v", err)
				}
				if info.Mode().Perm() != tt.perm {
					t.Errorf("File permissions mismatch: got %o, want %o", info.Mode().Perm(), tt.perm)
				}
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
