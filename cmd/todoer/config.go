// Package main provides the todoer CLI entrypoints and command helpers.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/inful/todoer/pkg/core"
)

// Config represents the configuration file structure
type Config struct {
	RootDir            string         `toml:"root_dir"`
	TemplateFile       string         `toml:"template_file"`
	Custom             map[string]any `toml:"custom_variables"`
	FrontmatterDateKey string         `toml:"frontmatter_date_key"`
	TodosHeader        string         `toml:"todos_header"`
	// BackupCount is how many previous versions of a file are kept in
	// ROOT/.backups/. nil means the default; 0 keeps no backups.
	BackupCount *int `toml:"backup_count"`
}

// loadConfig loads configuration from file, environment variables, and CLI flags
// Priority: CLI flags > environment variables > config file > defaults
//
// rootDirOverride is the --root-dir flag. It is applied before the
// configuration is validated, so the root the user asked for is checked
// exactly like the one from the config file.
func loadConfig(rootDirOverride string) (*Config, error) {
	config := &Config{}

	// Determine config file path
	configHome, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine config directory: %w", err)
	}
	configDir := filepath.Join(configHome, ConfigDirName)
	configPath := filepath.Join(configDir, ConfigFileName)

	// Load from config file if it exists
	if _, err := os.Stat(configPath); err == nil {
		if err := loadConfigFromFile(configPath, config); err != nil {
			return nil, err
		}
	}

	// Override with environment variables
	if rootDir := os.Getenv("TODOER_ROOT_DIR"); rootDir != "" {
		config.RootDir = expandPath(rootDir)
	}
	if templateFile := os.Getenv("TODOER_TEMPLATE_FILE"); templateFile != "" {
		config.TemplateFile = expandPath(templateFile)
	}
	if count := os.Getenv("TODOER_BACKUP_COUNT"); count != "" {
		n, err := strconv.Atoi(count)
		if err != nil {
			return nil, fmt.Errorf("TODOER_BACKUP_COUNT must be a number, got %q", count)
		}
		config.BackupCount = &n
	}

	// The CLI flag wins over the environment variable and the file
	if rootDirOverride != "" {
		config.RootDir = rootDirOverride
	}

	// Set defaults if not specified
	if config.RootDir == "" {
		config.RootDir = "."
	}
	if config.FrontmatterDateKey == "" {
		config.FrontmatterDateKey = "title"
	}
	if config.TodosHeader == "" {
		config.TodosHeader = core.TodosHeader
	}

	// Validate the final configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// backups returns how many previous versions of a file are kept. A config
// without the setting uses the default.
func (c *Config) backups() int {
	if c == nil || c.BackupCount == nil {
		return DefaultBackupCount
	}
	return *c.BackupCount
}

// loadConfigFromFile loads configuration from a TOML file
func loadConfigFromFile(configPath string, config *Config) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	if err := toml.Unmarshal(content, config); err != nil {
		return fmt.Errorf("failed to decode config file %s: %w", configPath, err)
	}

	// Expand paths in config
	if config.RootDir != "" {
		config.RootDir = expandPath(config.RootDir)
	}
	if config.TemplateFile != "" {
		config.TemplateFile = expandPath(config.TemplateFile)
	}

	return nil
}

// expandPath expands ~ to the user's home directory
func expandPath(path string) string {
	if path == "" {
		return path
	}

	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path // fallback to original path
		}
		return filepath.Join(homeDir, path[2:])
	}

	return path
}

// getConfigDir returns the appropriate config directory based on XDG or default
func getConfigDir() (string, error) {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return xdgConfigHome, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config"), nil
}
