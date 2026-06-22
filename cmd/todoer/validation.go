package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/inful/todoer/pkg/core"
)

// Validation errors
var (
	ErrInvalidPath      = errors.New("invalid file path")
	ErrSameSourceTarget = errors.New("source and target files cannot be the same")
	ErrInvalidDate      = errors.New("invalid date format")
	ErrInvalidConfig    = errors.New("invalid configuration")
	ErrConfigNotFound   = errors.New("configuration file not found")
	ErrPermissionDenied = errors.New("permission denied")
	ErrTemplateNotFound = errors.New("template file not found")
)

// validateFilePath validates a file path for security and correctness
func validateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: path cannot be empty", ErrInvalidPath)
	}

	// Check the raw input for directory traversal BEFORE filepath.Clean
	// resolves it. filepath.Clean("/foo/../../etc/passwd") → "/etc/passwd",
	// removing the ".." traces, so the check must happen on the original
	// string. We check at component boundaries (a substring check would
	// wrongly reject a legitimate filename like "..notes.md").
	if slices.Contains(strings.Split(filepath.ToSlash(path), "/"), "..") {
		return fmt.Errorf("%w: path contains directory traversal", ErrInvalidPath)
	}

	// Clean the path to normalize it after the traversal check.
	cleanPath := filepath.Clean(path)

	// Walk up the directory tree iteratively (bounded) to find the first existing
	// ancestor, ensuring the path is rooted in a real directory.
	const maxDepth = 50
	dir := filepath.Dir(cleanPath)
	for range maxDepth {
		if dir == "." || dir == "/" {
			break
		}
		info, err := os.Stat(dir)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("%w: parent path is not a directory", ErrInvalidPath)
			}
			break // found an existing valid directory ancestor
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("%w: cannot access directory: %v", ErrInvalidPath, err)
		}
		// Directory does not exist yet; check one level up.
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}

	return nil
}

// validateDateFormat validates date string format.
// Delegates to core.ValidateDate so there is one source of truth for the
// YYYY-MM-DD format. The error is wrapped with the public ErrInvalidDate
// sentinel so callers can use errors.Is to detect this case.
func validateDateFormat(date string) error {
	if date == "" {
		return nil // Empty date is valid (will use current date)
	}

	if err := core.ValidateDate(date); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidDate, err)
	}

	return nil
}

// validateProcessArgs validates arguments for the process command
func validateProcessArgs(sourceFile, targetFile, templateDate string) error {
	if err := validateFilePath(sourceFile); err != nil {
		return fmt.Errorf("invalid source file: %w", err)
	}

	if err := validateFilePath(targetFile); err != nil {
		return fmt.Errorf("invalid target file: %w", err)
	}

	// Check that source and target are different
	absSource, err := filepath.Abs(sourceFile)
	if err != nil {
		return fmt.Errorf("cannot resolve source file path: %w", err)
	}

	absTarget, err := filepath.Abs(targetFile)
	if err != nil {
		return fmt.Errorf("cannot resolve target file path: %w", err)
	}

	if absSource == absTarget {
		return ErrSameSourceTarget
	}

	if err := validateDateFormat(templateDate); err != nil {
		return fmt.Errorf("invalid template date: %w", err)
	}

	return nil
}

// validateConfig validates the configuration structure.
// It only observes configuration — it does not create directories or modify state.
func validateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("%w: config cannot be nil", ErrInvalidConfig)
	}

	if config.RootDir == "" {
		return fmt.Errorf("%w: root directory cannot be empty", ErrInvalidConfig)
	}

	// Validate root directory path
	if err := validateFilePath(config.RootDir); err != nil {
		return fmt.Errorf("invalid root directory '%s': %w", config.RootDir, err)
	}

	// Check if root directory exists and is accessible (if it exists)
	if info, err := os.Stat(config.RootDir); err != nil {
		if !os.IsNotExist(err) {
			if os.IsPermission(err) {
				return fmt.Errorf("%w: cannot access root directory '%s': %w", ErrPermissionDenied, config.RootDir, err)
			}
			return fmt.Errorf("%w: error accessing root directory '%s': %w", ErrInvalidConfig, config.RootDir, err)
		}
	} else if !info.IsDir() {
		return fmt.Errorf("%w: root path '%s' exists but is not a directory", ErrInvalidConfig, config.RootDir)
	}

	// Validate template file if specified
	if config.TemplateFile != "" {
		if err := validateFilePath(config.TemplateFile); err != nil {
			return fmt.Errorf("invalid template file '%s': %w", config.TemplateFile, err)
		}

		// Check if template file exists and is readable
		if info, err := os.Stat(config.TemplateFile); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("%w: template file '%s' does not exist", ErrTemplateNotFound, config.TemplateFile)
			}
			if os.IsPermission(err) {
				return fmt.Errorf("%w: cannot read template file '%s': %w", ErrPermissionDenied, config.TemplateFile, err)
			}
			return fmt.Errorf("%w: error accessing template file '%s': %w", ErrInvalidConfig, config.TemplateFile, err)
		} else if info.IsDir() {
			return fmt.Errorf("%w: template path '%s' is a directory, not a file", ErrInvalidConfig, config.TemplateFile)
		}
	}

	// Validate custom variables if present
	if err := validateCustomVariables(config.Custom); err != nil {
		return fmt.Errorf("%w: invalid custom variables: %w", ErrInvalidConfig, err)
	}

	return nil
}

// ensureDirectories creates any directories required by the configuration that do not yet exist.
func ensureDirectories(config *Config) error {
	if config == nil {
		return nil
	}
	if config.RootDir != "" {
		if err := os.MkdirAll(config.RootDir, 0o755); err != nil {
			return fmt.Errorf("%w: cannot create root directory '%s': %w", ErrInvalidConfig, config.RootDir, err)
		}
	}
	return nil
}

// validateCustomVariables validates the custom variables configuration.
// It delegates name and type checking to core.ValidateCustomVariables so both
// layers enforce exactly the same rules.
func validateCustomVariables(custom map[string]any) error {
	if custom == nil {
		return nil
	}
	return core.ValidateCustomVariables(custom)
}
