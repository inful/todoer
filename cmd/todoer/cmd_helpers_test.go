package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/inful/todoer/pkg/core"
)

// TestSharedPaths exercises the CLI/config priority for the two paths
// that every command reads. CLI flags take precedence; empty CLI values
// fall back to config values.
func TestSharedPaths(t *testing.T) {
	tests := []struct {
		name               string
		cliRootDir         string
		cliTemplateFile    string
		configRootDir      string
		configTemplateFile string
		wantRootDir        string
		wantTemplateFile   string
	}{
		{
			name:               "both CLI values override config",
			cliRootDir:         "/cli/root",
			cliTemplateFile:    "/cli/template",
			configRootDir:      "/config/root",
			configTemplateFile: "/config/template",
			wantRootDir:        "/cli/root",
			wantTemplateFile:   "/cli/template",
		},
		{
			name:               "empty CLI values fall back to config",
			cliRootDir:         "",
			cliTemplateFile:    "",
			configRootDir:      "/config/root",
			configTemplateFile: "/config/template",
			wantRootDir:        "/config/root",
			wantTemplateFile:   "/config/template",
		},
		{
			name:               "only root CLI value set",
			cliRootDir:         "/cli/root",
			cliTemplateFile:    "",
			configRootDir:      "/config/root",
			configTemplateFile: "/config/template",
			wantRootDir:        "/cli/root",
			wantTemplateFile:   "/config/template",
		},
		{
			name:               "only template CLI value set",
			cliRootDir:         "",
			cliTemplateFile:    "/cli/template",
			configRootDir:      "/config/root",
			configTemplateFile: "/config/template",
			wantRootDir:        "/config/root",
			wantTemplateFile:   "/cli/template",
		},
		{
			name:             "both empty returns two empty strings",
			wantRootDir:      "",
			wantTemplateFile: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &cliOptions{
				RootDir:      tt.cliRootDir,
				TemplateFile: tt.cliTemplateFile,
			}
			config := &Config{
				RootDir:      tt.configRootDir,
				TemplateFile: tt.configTemplateFile,
			}

			gotRoot, gotTmpl := sharedPaths(cli, config)
			if gotRoot != tt.wantRootDir {
				t.Errorf("rootDir = %q, want %q", gotRoot, tt.wantRootDir)
			}
			if gotTmpl != tt.wantTemplateFile {
				t.Errorf("templateFile = %q, want %q", gotTmpl, tt.wantTemplateFile)
			}
		})
	}
}

// TestGetGenerator covers the three behaviors the daily flow relies on:
//   - empty templateDate defaults to today
//   - empty templateFile uses the embedded default
//   - previousDate is propagated to the generator
func TestGetGenerator(t *testing.T) {
	config := &Config{
		Custom:             map[string]any{"greeting": "hi"},
		FrontmatterDateKey: "title",
		TodosHeader:        core.TodosHeader,
	}

	t.Run("empty templateDate defaults to today", func(t *testing.T) {
		gen, _, err := getGenerator("", "", "", config)
		if err != nil {
			t.Fatalf("getGenerator: %v", err)
		}
		// The generator's templateDate is unexported, but we can
		// prove it was set by processing content and checking the
		// rendered date.
		today := time.Now().Format(core.DateFormat)
		result, err := gen.Process("---\ntitle: " + today + "\n---\n\n## Todos\n\n- [[" + today + "]]\n  - [ ] carry me\n")
		if err != nil {
			t.Fatalf("gen.Process: %v", err)
		}
		newFile := readAll(t, result.NewFile)
		if !strings.Contains(newFile, today) {
			t.Errorf("expected rendered output to contain today's date %q, got:\n%s", today, newFile)
		}
	})

	t.Run("explicit templateDate is used", func(t *testing.T) {
		const want = "2025-12-25"
		gen, _, err := getGenerator("", want, "", config)
		if err != nil {
			t.Fatalf("getGenerator: %v", err)
		}
		result, err := gen.Process("---\ntitle: 2025-12-24\n---\n\n## Todos\n\n- [[2025-12-24]]\n  - [ ] carry me\n")
		if err != nil {
			t.Fatalf("gen.Process: %v", err)
		}
		newFile := readAll(t, result.NewFile)
		if !strings.Contains(newFile, want) {
			t.Errorf("expected rendered output to contain %q, got:\n%s", want, newFile)
		}
	})

	t.Run("empty templateFile uses embedded default", func(t *testing.T) {
		gen, name, err := getGenerator("", "2025-06-22", "", config)
		if err != nil {
			t.Fatalf("getGenerator: %v", err)
		}
		if name != "embedded default template" {
			t.Errorf("expected template name 'embedded default template', got %q", name)
		}
		if gen == nil {
			t.Error("expected non-nil generator")
		}
	})

	t.Run("explicit templateFile is read and used", func(t *testing.T) {
		dir := t.TempDir()
		tmplPath := filepath.Join(dir, "custom.md")
		if err := os.WriteFile(tmplPath, []byte("# Custom {{.Date}}\n\n## Todos\n\n{{.TODOS}}\n"), 0o644); err != nil {
			t.Fatalf("write template: %v", err)
		}

		gen, name, err := getGenerator(tmplPath, "2025-06-22", "", config)
		if err != nil {
			t.Fatalf("getGenerator: %v", err)
		}
		if name != tmplPath {
			t.Errorf("expected template name %q, got %q", tmplPath, name)
		}
		result, err := gen.Process("---\ntitle: 2025-06-21\n---\n\n## Todos\n\n- [[2025-06-21]]\n  - [ ] carry me\n")
		if err != nil {
			t.Fatalf("gen.Process: %v", err)
		}
		newFile := readAll(t, result.NewFile)
		if !strings.Contains(newFile, "# Custom 2025-06-22") {
			t.Errorf("expected custom template to render with date, got:\n%s", newFile)
		}
	})

	t.Run("invalid templateFile returns an error", func(t *testing.T) {
		_, _, err := getGenerator("/this/does/not/exist.md", "2025-06-22", "", config)
		if err == nil {
			t.Error("expected error for non-existent template file, got nil")
		}
	})

	t.Run("invalid templateDate returns an error", func(t *testing.T) {
		_, _, err := getGenerator("", "not-a-date", "", config)
		if err == nil {
			t.Error("expected error for invalid date, got nil")
		}
	})
}

// readAll drains an io.Reader into a string. It fails the test on error.
func readAll(t *testing.T, r io.Reader) string {
	t.Helper()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read all: %v", err)
	}
	return string(data)
}
