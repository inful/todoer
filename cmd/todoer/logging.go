package main

import (
	"io"
	"log"
	"os"
)

// Logger for debug and verbose output
var debugLogger *log.Logger

func init() {
	// Initialize debug logger - disabled by default
	debugLogger = log.New(io.Discard, "DEBUG: ", log.LstdFlags|log.Lshortfile)
}

// enableDebugLogging enables debug logging to stderr
func enableDebugLogging() {
	debugLogger.SetOutput(os.Stderr)
}

// logDebug logs a debug message if debug logging is enabled
func logDebug(format string, args ...interface{}) {
	debugLogger.Printf(format, args...)
}

// logInfo logs an info message to stderr
func logInfo(format string, args ...interface{}) {
	log.Printf("INFO: "+format, args...)
}

// logError logs an error message to stderr
func logError(format string, args ...interface{}) {
	log.Printf("ERROR: "+format, args...)
}
