package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"todoer/pkg/core"
)

// Validation errors
var (
	ErrInvalidPath      = errors.New("invalid file path")
	ErrSameSourceTarget = errors.New("source and target files cannot be the same")
	ErrInvalidDate      = errors.New("invalid date format")
	ErrInvalidConfig    = errors.New("invalid configuration")
)

// validateFilePath validates a file path for security and correctness
func validateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: path cannot be empty", ErrInvalidPath)
	}

	// Clean the path to resolve any .. or . components
	cleanPath := filepath.Clean(path)

	// Check for potentially dangerous paths
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("%w: path contains directory traversal", ErrInvalidPath)
	}

	// Check if the directory portion exists or can be created
	dir := filepath.Dir(cleanPath)
	if dir != "." && dir != "/" {
		if info, err := os.Stat(dir); err != nil {
			if os.IsNotExist(err) {
				// Check if we can potentially create the directory
				parent := filepath.Dir(dir)
				if parent != dir { // Avoid infinite recursion
					if err := validateFilePath(parent); err != nil {
						return fmt.Errorf("%w: cannot access parent directory", ErrInvalidPath)
					}
				}
			} else {
				return fmt.Errorf("%w: cannot access directory: %v", ErrInvalidPath, err)
			}
		} else if !info.IsDir() {
			return fmt.Errorf("%w: parent path is not a directory", ErrInvalidPath)
		}
	}

	return nil
}

// validateDateFormat validates date string format
func validateDateFormat(date string) error {
	if date == "" {
		return nil // Empty date is valid (will use current date)
	}

	if _, err := time.Parse(core.DateFormat, date); err != nil {
		return fmt.Errorf("%w: expected format YYYY-MM-DD, got %s", ErrInvalidDate, date)
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

// validateConfig validates the configuration structure
func validateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("%w: config cannot be nil", ErrInvalidConfig)
	}

	if config.RootDir == "" {
		return fmt.Errorf("%w: root directory cannot be empty", ErrInvalidConfig)
	}

	// Validate root directory path
	if err := validateFilePath(config.RootDir); err != nil {
		return fmt.Errorf("invalid root directory: %w", err)
	}

	// Validate template file if specified
	if config.TemplateFile != "" {
		if err := validateFilePath(config.TemplateFile); err != nil {
			return fmt.Errorf("invalid template file: %w", err)
		}
	}

	return nil
}
