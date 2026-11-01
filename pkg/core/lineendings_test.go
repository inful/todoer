package core

import (
	"strings"
	"testing"
)

// TestMixedLineEndings pins that a journal mixing LF and CRLF is read and
// written back the way it came in. A single Windows line ending in an
// otherwise LF file used to make every command fail with "expected blank
// line after header".
func TestMixedLineEndings(t *testing.T) {
	tests := []struct {
		name    string
		content string
		// wantLF is the ending the written file must use throughout.
		wantLF bool
	}{
		{
			name: "one stray CRLF among LF lines",
			content: "---\ntitle: 2026-10-22\n---\n\n## Todos\n\n- [[2026-10-22]]\n  - [ ] task\n\n" +
				"## Notes\n\none stray\r\nwindows line\nand an lf line\n",
			wantLF: true,
		},
		{
			name: "one stray LF among CRLF lines",
			content: "---\r\ntitle: 2026-10-22\r\n---\r\n\r\n## Todos\r\n\r\n- [[2026-10-22]]\r\n  - [ ] task\r\n\r\n" +
				"## Notes\r\n\r\none stray\nlf line\r\n",
			wantLF: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before, section, after, err := ExtractTodosSectionWithHeader(tt.content, TodosHeader)
			if err != nil {
				t.Fatalf("extract: %v", err)
			}
			if !strings.Contains(section, "- [ ] task") {
				t.Fatalf("todos section not found, got %q", section)
			}

			journal, err := ParseTodosSection(section)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(journal.Days) != 1 || len(journal.Days[0].Items) != 1 {
				t.Fatalf("expected one day with one item, got %+v", journal.Days)
			}
			if item := journal.Days[0].Items[0]; item.Text != "task" {
				t.Errorf("item text = %q, want %q", item.Text, "task")
			}

			// Writing it back keeps the file's dominant ending: an LF
			// file stays LF, a CRLF file is normalised to CRLF.
			out := MatchLineEndings(before+JournalToString(journal)+after, tt.content)
			if tt.wantLF {
				// Same mix of endings as it came in with.
				if crlf(out) != crlf(tt.content) || bareLF(out) != bareLF(tt.content) {
					t.Errorf("line endings changed:\ngot:  %q\nwant: %q", out, tt.content)
				}
			} else if bareLF(out) != 0 {
				t.Errorf("expected CRLF throughout, found %d bare LF in %q", bareLF(out), out)
			}
			if !strings.Contains(out, "- [ ] task") || !strings.Contains(out, "one stray") {
				t.Errorf("content lost:\n%q", out)
			}
		})
	}
}

// crlf counts the CRLF line endings in s.
func crlf(s string) int { return strings.Count(s, "\r\n") }

// bareLF counts the LF line endings that are not part of a CRLF.
func bareLF(s string) int { return strings.Count(s, "\n") - crlf(s) }

// TestMatchLineEndingsDominant pins that a file with a stray CRLF is not
// rewritten wholesale to CRLF.
func TestMatchLineEndingsDominant(t *testing.T) {
	lf := "a\nb\nc\n"
	if got := MatchLineEndings(lf, lf+"\r\n"); got != lf {
		t.Errorf("an LF file with one stray CRLF must stay LF, got %q", got)
	}

	crlf := "a\r\nb\r\nc\r\n"
	if got := MatchLineEndings(crlf, crlf+"\n"); got != crlf {
		t.Errorf("a CRLF file with one stray LF must stay CRLF, got %q", got)
	}
}
