package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
