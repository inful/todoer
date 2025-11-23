package main

import (
	"fmt"
	"log"
	"os"
)

// OutputMode controls verbosity of logging.
type OutputMode int

const (
	ModeNormal OutputMode = iota
	ModeQuiet
	ModeDebug
)

// Logger encapsulates log behavior based on OutputMode.
type Logger struct {
	mode OutputMode
}

// NewLogger creates a new Logger with the given mode.
func NewLogger(mode OutputMode) *Logger {
	return &Logger{mode: mode}
}

// WithMode returns a new Logger with updated mode.
func (l *Logger) WithMode(mode OutputMode) *Logger {
	return &Logger{mode: mode}
}

// Info logs informational messages unless in quiet mode.
func (l *Logger) Info(format string, args ...interface{}) {
	if l.mode == ModeQuiet {
		return
	}
	fmt.Fprintf(os.Stderr, "INFO: "+format+"\n", args...)
}

// Debug logs debug messages only in debug mode.
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.mode != ModeDebug {
		return
	}
	log.Printf("DEBUG: "+format, args...)
}

// Error always logs errors.
func (l *Logger) Error(format string, args ...interface{}) {
	log.Printf("ERROR: "+format, args...)
}
