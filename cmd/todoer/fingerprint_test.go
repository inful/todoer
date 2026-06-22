package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/inful/todoer/pkg/core"
)

func TestFingerprintEnabled_Default(t *testing.T) {
	// TODOER_FINGERPRINT is unset by default in tests; the helper
	// should report false.
	if err := os.Unsetenv(fingerprintEnabledEnv); err != nil {
		t.Fatalf("unsetenv: %v", err)
	}
	if fingerprintEnabled() {
		t.Fatalf("expected fingerprintEnabled to be false by default")
	}
}

func TestFingerprintEnabled_TruthyValues(t *testing.T) {
	// strconv.ParseBool accepts: 1, t, T, TRUE, true, True,
	// 0, f, F, FALSE, false, False. We expect the spike to
	// turn on for any of the truthy spellings and to stay
	// off for the falsy ones and for garbage.
	for _, value := range []string{"1", "t", "T", "true", "TRUE", "True"} {
		t.Setenv(fingerprintEnabledEnv, value)
		if !fingerprintEnabled() {
			t.Fatalf("expected fingerprintEnabled to be true for %q", value)
		}
	}
	for _, value := range []string{"0", "f", "F", "false", "FALSE", "no", "", "yes-please"} {
		t.Setenv(fingerprintEnabledEnv, value)
		if fingerprintEnabled() {
			t.Fatalf("expected fingerprintEnabled to be false for %q", value)
		}
	}
}

func TestFingerprintWriteEnabled_Default(t *testing.T) {
	// TODOER_FINGERPRINT_WRITE is unset by default. The default
	// must be "write enabled" so that turning the spike on with
	// TODOER_FINGERPRINT=1 retains the existing behavior.
	if err := os.Unsetenv("TODOER_FINGERPRINT_WRITE"); err != nil {
		t.Fatalf("unsetenv: %v", err)
	}
	if !fingerprintWriteEnabled() {
		t.Fatalf("expected fingerprintWriteEnabled to be true by default")
	}
}

func TestFingerprintWriteEnabled_TruthyValues(t *testing.T) {
	for _, value := range []string{"1", "t", "T", "true", "TRUE", "True"} {
		t.Setenv("TODOER_FINGERPRINT_WRITE", value)
		if !fingerprintWriteEnabled() {
			t.Fatalf("expected fingerprintWriteEnabled to be true for %q", value)
		}
	}
	for _, value := range []string{"0", "f", "F", "false", "FALSE"} {
		t.Setenv("TODOER_FINGERPRINT_WRITE", value)
		if fingerprintWriteEnabled() {
			t.Fatalf("expected fingerprintWriteEnabled to be false for %q", value)
		}
	}
	// Unset and unparseable values fall back to the documented default
	// (write enabled).
	t.Setenv("TODOER_FINGERPRINT_WRITE", "")
	if !fingerprintWriteEnabled() {
		t.Fatalf("expected fingerprintWriteEnabled to be true for unset value (default on)")
	}
	t.Setenv("TODOER_FINGERPRINT_WRITE", "garbage")
	if !fingerprintWriteEnabled() {
		t.Fatalf("expected fingerprintWriteEnabled to be true for unparseable value (default on)")
	}
}

func TestComputeFingerprint_Deterministic(t *testing.T) {
	// mdfp fingerprints a canonical virtual document of
	// frontmatter (with the fingerprint field stripped) plus body.
	// The test inputs need a recognizable body to fingerprint.
	content := []byte("# hello world\n\nSome body.\n")
	got, err := computeFingerprint(content)
	if err != nil {
		t.Fatalf("computeFingerprint: %v", err)
	}
	if len(got) != 64 {
		t.Fatalf("expected 64-char hex SHA-256, got %d chars: %s", len(got), got)
	}
	got2, err := computeFingerprint(content)
	if err != nil {
		t.Fatalf("computeFingerprint (second call): %v", err)
	}
	if got != got2 {
		t.Fatalf("expected deterministic fingerprint, got %q and %q", got, got2)
	}
	// Different content should yield a different fingerprint.
	got3, err := computeFingerprint([]byte("# hello world!\n\nSome body.\n"))
	if err != nil {
		t.Fatalf("computeFingerprint (third call): %v", err)
	}
	if got == got3 {
		t.Fatalf("expected different content to yield different fingerprint")
	}
}

func TestComputeFingerprint_FrontmatterChangesHash(t *testing.T) {
	// mdfp includes the frontmatter in the hash, with only the
	// 'fingerprint' field stripped. Two documents with the same
	// body but different frontmatter metadata must therefore
	// produce different fingerprints. This pins the
	// canonicalization contract — if mdfp ever changes to
	// exclude frontmatter, this test will fail.
	body := "\n# hello world\n\nSome body.\n"
	withTitleA := []byte("---\ntitle: A\n---\n" + body)
	withTitleB := []byte("---\ntitle: B\n---\n" + body)
	fpA, err := computeFingerprint(withTitleA)
	if err != nil {
		t.Fatalf("computeFingerprint (title A): %v", err)
	}
	fpB, err := computeFingerprint(withTitleB)
	if err != nil {
		t.Fatalf("computeFingerprint (title B): %v", err)
	}
	if fpA == fpB {
		t.Fatalf("expected different frontmatter to change the hash, both = %q", fpA)
	}
	// Same frontmatter should still produce the same hash.
	fpA2, err := computeFingerprint([]byte("---\ntitle: A\n---\n" + body))
	if err != nil {
		t.Fatalf("computeFingerprint (title A again): %v", err)
	}
	if fpA != fpA2 {
		t.Fatalf("expected deterministic fingerprint for same content, got %q and %q", fpA, fpA2)
	}
}

func TestCheckFingerprintMismatch_NoPriorFingerprint(_ *testing.T) {
	// Existing target has no frontmatter fingerprint yet. The check
	// should be a silent no-op (treat as fresh sync).
	logger := NewLogger(ModeQuiet)
	existing := []byte("# Just a heading\n\nNo frontmatter.\n")
	checkFingerprintMismatch(existing, []byte("source content"), "/tmp/whatever", logger)
}

func TestCheckFingerprintMismatch_Matching(t *testing.T) {
	source := []byte("# the source\n\nbody content.\n")
	fp, err := computeFingerprint(source)
	if err != nil {
		t.Fatalf("computeFingerprint: %v", err)
	}
	existing := []byte("---\ntitle: x\nfingerprint: " + fp + "\n---\n\n# Body\n")
	logger := NewLogger(ModeQuiet)
	// Same source -> no mismatch log; we just check it does not panic.
	checkFingerprintMismatch(existing, source, "/tmp/whatever", logger)
}

func TestCheckFingerprintMismatch_Differing(_ *testing.T) {
	// Existing target records a fingerprint for some other source
	// content. The current source differs; the check should log a
	// mismatch and not fail.
	existing := []byte("---\ntitle: x\nfingerprint: deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef\n---\n\n# Body\n")
	logger := NewLogger(ModeQuiet)
	checkFingerprintMismatch(existing, []byte("# the actual source\n\nbody.\n"), "/tmp/whatever", logger)
}

func TestProcessJournal_FingerprintMismatchForcesReMerge(t *testing.T) {
	// End-to-end: with the toggle on, the second run with a modified
	// source should still merge cleanly. The fingerprint mismatch is a
	// hint, not a failure.
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)

	// Source with one carryover item.
	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] carry me\n\n## Notes\n")

	// Pre-existing target with a stale fingerprint and a fresh
	// today-only item, so the merge will both keep the existing
	// item and detect a fingerprint mismatch.
	preExistingTarget := "---\ndate: " + today + "\nfingerprint: 0000000000000000000000000000000000000000000000000000000000000000\n---\n\n# J\n\n## Todos\n\n- [[" + today + "]]\n  - [ ] already here\n\n## Notes\n"
	createTestFile(t, target, preExistingTarget)

	t.Setenv(fingerprintEnabledEnv, "1")

	logger := NewLogger(ModeQuiet)
	if err := processJournal(ProcessOptions{
		SourceFile:    source,
		TargetFile:    target,
		TemplateDate:  today,
		MergeIfExists: true,
	}, config, logger); err != nil {
		t.Fatalf("processJournal: %v", err)
	}

	after, _ := os.ReadFile(target)
	content := string(after)
	if !strings.Contains(content, "carry me") {
		t.Fatalf("expected the carryover item to be merged in, got:\n%s", content)
	}
	if !strings.Contains(content, "already here") {
		t.Fatalf("expected the pre-existing today item to be preserved, got:\n%s", content)
	}
}

func TestProcessJournal_RecordsFingerprintWhenEnabled(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)

	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] carry me\n\n## Notes\n")

	t.Setenv(fingerprintEnabledEnv, "1")

	logger := NewLogger(ModeQuiet)
	if err := processJournal(ProcessOptions{
		SourceFile:    source,
		TargetFile:    target,
		TemplateDate:  today,
		MergeIfExists: true,
	}, config, logger); err != nil {
		t.Fatalf("processJournal: %v", err)
	}

	after, _ := os.ReadFile(target)
	content := string(after)
	if !strings.Contains(content, "fingerprint:") {
		t.Fatalf("expected fingerprint key in target frontmatter, got:\n%s", content)
	}
	if strings.Contains(content, "todoer_source_fingerprint") {
		t.Fatalf("expected old 'todoer_source_fingerprint' field name to be gone, got:\n%s", content)
	}
	// The recorded fingerprint should be a 64-char hex SHA-256.
	before, afterKey, found := strings.Cut(content, "fingerprint: ")
	if !found {
		t.Fatalf("expected 'fingerprint: ' key, got:\n%s", content)
	}
	_ = before
	end := strings.IndexByte(afterKey, '\n')
	if end < 0 {
		t.Fatalf("expected newline after fingerprint value, got:\n%s", content)
	}
	value := strings.TrimSpace(afterKey[:end])
	if len(value) != 64 {
		t.Fatalf("expected 64-char hex fingerprint, got %d chars: %q", len(value), value)
	}
}

func TestProcessJournal_NoFingerprintWhenDisabled(t *testing.T) {
	// Without TODOER_FINGERPRINT, the spike is invisible: the target
	// does not receive a fingerprint key.
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)

	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] carry me\n\n## Notes\n")

	// Make sure the env var is unset.
	if err := os.Unsetenv(fingerprintEnabledEnv); err != nil {
		t.Fatalf("unsetenv: %v", err)
	}

	logger := NewLogger(ModeQuiet)
	if err := processJournal(ProcessOptions{
		SourceFile:    source,
		TargetFile:    target,
		TemplateDate:  today,
		MergeIfExists: true,
	}, config, logger); err != nil {
		t.Fatalf("processJournal: %v", err)
	}

	after, _ := os.ReadFile(target)
	if strings.Contains(string(after), "todoer_source_fingerprint") {
		t.Fatalf("expected no old todoer_source_fingerprint key when toggle is off, got:\n%s", string(after))
	}
	if strings.Contains(string(after), "fingerprint:") {
		t.Fatalf("expected no fingerprint key when toggle is off, got:\n%s", string(after))
	}
}

func TestProcessJournal_NoFingerprintWriteWhenSuppressed(t *testing.T) {
	// With TODOER_FINGERPRINT=1 (spike on) and TODOER_FINGERPRINT_WRITE=0
	// (write side suppressed), the target does not receive a fingerprint
	// key. The read side (mismatch detection) is still active.
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)

	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] carry me\n\n## Notes\n")

	t.Setenv(fingerprintEnabledEnv, "1")
	t.Setenv(fingerprintWriteEnv, "0")

	logger := NewLogger(ModeQuiet)
	if err := processJournal(ProcessOptions{
		SourceFile:    source,
		TargetFile:    target,
		TemplateDate:  today,
		MergeIfExists: true,
	}, config, logger); err != nil {
		t.Fatalf("processJournal: %v", err)
	}

	after, _ := os.ReadFile(target)
	if strings.Contains(string(after), "todoer_source_fingerprint") {
		t.Fatalf("expected no old todoer_source_fingerprint key when write is suppressed, got:\n%s", string(after))
	}
	if strings.Contains(string(after), "fingerprint:") {
		t.Fatalf("expected no fingerprint key when write is suppressed, got:\n%s", string(after))
	}
}
