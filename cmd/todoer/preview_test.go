package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/inful/todoer/pkg/core"
)

// TestCmdPreview covers the input combinations of the `preview` subcommand:
//   - templateFile selection (CLI / config-dir / embedded default)
//   - todos source precedence (string > file > default sample)
//   - custom variables precedence (CLI JSON > config)
//   - default date is today when none given
func TestCmdPreview(t *testing.T) {
	const tmplContent = "# {{.Date}}\n\n## Todos\n\n{{.TODOS}}\n"

	newTmplFile := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		p := filepath.Join(dir, "tmpl.md")
		if err := os.WriteFile(p, []byte(tmplContent), 0o644); err != nil {
			t.Fatalf("write template: %v", err)
		}
		return p
	}

	t.Run("explicit template file is used", func(t *testing.T) {
		tmpl := newTmplFile(t)
		out := captureStdout(t, func() error {
			return cmdPreview(tmpl, "2025-06-22", "", "", "", &Config{TodosHeader: core.TodosHeader})
		})
		if !strings.Contains(out, "2025-06-22") {
			t.Errorf("expected rendered output to contain date, got:\n%s", out)
		}
	})

	t.Run("empty date defaults to today", func(t *testing.T) {
		tmpl := newTmplFile(t)
		out := captureStdout(t, func() error {
			return cmdPreview(tmpl, "", "", "", "", &Config{TodosHeader: core.TodosHeader})
		})
		// Today's date should appear in the rendered template title.
		// The date appears in two places (title + frontmatter-ish).
		if !strings.Contains(out, time.Now().Format(core.DateFormat)) {
			t.Errorf("expected rendered output to contain today's date, got:\n%s", out)
		}
	})

	t.Run("todos string is used when provided", func(t *testing.T) {
		tmpl := newTmplFile(t)
		const todos = "- [[2025-06-20]]\n  - [ ] inline string task\n"
		out := captureStdout(t, func() error {
			return cmdPreview(tmpl, "2025-06-22", "", todos, "", &Config{TodosHeader: core.TodosHeader})
		})
		if !strings.Contains(out, "inline string task") {
			t.Errorf("expected inline string task in output, got:\n%s", out)
		}
	})

	t.Run("todos file is read when no string is given", func(t *testing.T) {
		tmpl := newTmplFile(t)
		dir := t.TempDir()
		todosPath := filepath.Join(dir, "todos.md")
		const todos = "- [[2025-06-20]]\n  - [ ] file sourced task\n"
		if err := os.WriteFile(todosPath, []byte(todos), 0o644); err != nil {
			t.Fatalf("write todos: %v", err)
		}
		out := captureStdout(t, func() error {
			return cmdPreview(tmpl, "2025-06-22", todosPath, "", "", &Config{TodosHeader: core.TodosHeader})
		})
		if !strings.Contains(out, "file sourced task") {
			t.Errorf("expected file-sourced task in output, got:\n%s", out)
		}
	})

	t.Run("todos string overrides todos file", func(t *testing.T) {
		tmpl := newTmplFile(t)
		dir := t.TempDir()
		todosPath := filepath.Join(dir, "todos.md")
		if err := os.WriteFile(todosPath, []byte("- [[2025-06-20]]\n  - [ ] from file\n"), 0o644); err != nil {
			t.Fatalf("write todos: %v", err)
		}
		out := captureStdout(t, func() error {
			return cmdPreview(tmpl, "2025-06-22", todosPath, "- [[2025-06-20]]\n  - [ ] from string\n", "", &Config{TodosHeader: core.TodosHeader})
		})
		if !strings.Contains(out, "from string") {
			t.Errorf("expected string override in output, got:\n%s", out)
		}
		if strings.Contains(out, "from file") {
			t.Errorf("did not expect file content when string override is given, got:\n%s", out)
		}
	})

	t.Run("default sample todos used when no source given", func(t *testing.T) {
		tmpl := newTmplFile(t)
		out := captureStdout(t, func() error {
			return cmdPreview(tmpl, "2025-06-22", "", "", "", &Config{TodosHeader: core.TodosHeader})
		})
		// The default sample in preview.go includes these strings.
		if !strings.Contains(out, "Task from Friday") {
			t.Errorf("expected default sample content in output, got:\n%s", out)
		}
	})

	t.Run("custom vars JSON overrides config.Custom", func(t *testing.T) {
		const customTmpl = "# {{.Date}}\n{{with .Custom.greeting}}Greeting: {{.}}{{end}}\n\n## Todos\n\n{{.TODOS}}\n"
		dir := t.TempDir()
		tmplPath := filepath.Join(dir, "tmpl.md")
		if err := os.WriteFile(tmplPath, []byte(customTmpl), 0o644); err != nil {
			t.Fatalf("write template: %v", err)
		}

		cfg := &Config{
			TodosHeader: core.TodosHeader,
			Custom:      map[string]any{"greeting": "from-config"},
		}
		out := captureStdout(t, func() error {
			return cmdPreview(tmplPath, "2025-06-22", "", "- [[2025-06-22]]\n  - [ ] task\n", `{"greeting":"from-cli"}`, cfg)
		})
		if !strings.Contains(out, "Greeting: from-cli") {
			t.Errorf("expected CLI custom var to win, got:\n%s", out)
		}
		if strings.Contains(out, "from-config") {
			t.Errorf("did not expect config custom var when CLI override given, got:\n%s", out)
		}
	})

	t.Run("config custom vars used when CLI custom vars empty", func(t *testing.T) {
		dir := t.TempDir()
		tmplPath := filepath.Join(dir, "tmpl.md")
		const todosTmpl = "# {{.Date}}\n{{with .Custom.greeting}}Greeting: {{.}}{{end}}\n\n## Todos\n\n{{.TODOS}}\n"
		if err := os.WriteFile(tmplPath, []byte(todosTmpl), 0o644); err != nil {
			t.Fatalf("write template: %v", err)
		}

		cfg := &Config{
			TodosHeader: core.TodosHeader,
			Custom:      map[string]any{"greeting": "from-config"},
		}
		out := captureStdout(t, func() error {
			return cmdPreview(tmplPath, "2025-06-22", "", "- [[2025-06-22]]\n  - [ ] task\n", "", cfg)
		})
		if !strings.Contains(out, "Greeting: from-config") {
			t.Errorf("expected config custom var to be used, got:\n%s", out)
		}
	})

	t.Run("invalid custom vars JSON returns an error", func(t *testing.T) {
		tmpl := newTmplFile(t)
		err := cmdPreview(tmpl, "2025-06-22", "", "", "not-json", &Config{TodosHeader: core.TodosHeader})
		if err == nil {
			t.Error("expected error for invalid custom vars JSON, got nil")
		}
	})

	t.Run("non-existent todos file returns an error", func(t *testing.T) {
		tmpl := newTmplFile(t)
		err := cmdPreview(tmpl, "2025-06-22", "/does/not/exist.md", "", "", &Config{TodosHeader: core.TodosHeader})
		if err == nil {
			t.Error("expected error for non-existent todos file, got nil")
		}
	})
}

// captureStdout redirects os.Stdout for the duration of fn, returning
// everything written. Used because cmdPreview writes via fmt.Println.
func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	errCh := make(chan error, 1)
	var buf bytes.Buffer
	go func() {
		_, copyErr := io.Copy(&buf, r)
		errCh <- copyErr
		_ = r.Close()
	}()

	fnErr := fn()
	_ = w.Close()
	os.Stdout = orig
	if copyErr := <-errCh; copyErr != nil {
		t.Fatalf("copy stdout: %v", copyErr)
	}
	if fnErr != nil {
		t.Fatalf("function under test: %v", fnErr)
	}
	return buf.String()
}
