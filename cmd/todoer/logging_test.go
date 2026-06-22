package main

import (
	"testing"
)

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
