package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config represents the configuration file structure
type Config struct {
	RootDir            string                 `toml:"root_dir"`
	TemplateFile       string                 `toml:"template_file"`
	Custom             map[string]interface{} `toml:"custom_variables"`
	FrontmatterDateKey string                 `toml:"frontmatter_date_key"`
	TodosHeader        string                 `toml:"todos_header"`
}

// loadConfig loads configuration from file, environment variables, and CLI flags
// Priority: CLI flags > environment variables > config file > defaults
func loadConfig() (*Config, error) {
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
		// config loaded from file; logging omitted to keep config package decoupled
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

	// Set defaults if not specified
	if config.RootDir == "" {
		config.RootDir = "."
	}
	if config.FrontmatterDateKey == "" {
		config.FrontmatterDateKey = "title"
	}
	if config.TodosHeader == "" {
		config.TodosHeader = "## Todos"
	}

	// Validate the final configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
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
