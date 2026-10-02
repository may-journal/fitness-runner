package render

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func plain() Palette { return Palette{} }

// assertLineWidths checks every line of out is exactly width runes wide.
func assertLineWidths(t *testing.T, out string, width int) {
	t.Helper()
	for i, line := range strings.Split(out, "\n") {
		if got := utf8.RuneCountInString(line); got != width {
			t.Errorf("line %d width = %d, want %d: %q", i, got, width, line)
		}
	}
}

func TestTableGeometryAt80(t *testing.T) {
	rows := []Row{
		{Name: "read-repo-first", Ok: true, FilesChecked: 0, Ms: 7},
		{Name: "markdown-filename-kebab-case", Ok: false, FilesChecked: 49, Ms: 6,
			Errors: []string{"some/path.md: filename must be kebab-case"}},
	}
	out := Table(rows, 80, plain())
	assertLineWidths(t, out, 80)
	wants := []struct{ text, why string }{
		{"│ markdown-filename-kebab-c… │", "long name must truncate with ellipsis at 26 chars"},
		{"[markdown-filename-kebab-case] Please fix these items:", "error block header missing"},
		{"  ✖ some/path.md: filename must be kebab-case", "error bullet missing"},
		{"│ read-repo-first            │ passed   │ 0      │ 7ms", "row cells misaligned"},
	}
	for _, w := range wants {
		if !strings.Contains(out, w.text) {
			t.Error(w.why)
		}
	}
}

func TestTableNarrowTerminalClampsTimeColumn(t *testing.T) {
	out := Table([]Row{{Name: "x", Ok: true, FilesChecked: 1, Ms: 1}}, 40, plain())
	first := strings.Split(out, "\n")[0]
	// widths 28+10+8+10 plus 5 border chars
	if got := utf8.RuneCountInString(first); got != 61 {
		t.Fatalf("clamped width = %d, want 61", got)
	}
}

func TestFilesDashWhenNegative(t *testing.T) {
	out := Table([]Row{{Name: "x", Ok: true, FilesChecked: -1, Ms: 3}}, 80, plain())
	if !strings.Contains(out, "│ -      │") {
		t.Error("negative filesChecked must render as -")
	}
}

func TestErrorBlockWraps(t *testing.T) {
	long := strings.Repeat("word ", 30) + "end"
	out := Table([]Row{{Name: "x", Ok: false, FilesChecked: 1, Ms: 1, Errors: []string{long}}}, 80, plain())
	assertLineWidths(t, out, 80)
	if !strings.Contains(out, "│   ✖ word") {
		t.Error("first wrapped line must keep the bullet indent")
	}
}

func TestTotalLine(t *testing.T) {
	green := TotalLine(21, 0, 393, 1458, plain())
	if green != "✓ All 21 checks passed · 393 files scanned · 1458ms" {
		t.Fatalf("green: got %q", green)
	}
	failing := TotalLine(20, 1, 393, 1458, plain())
	if failing != "✗ 20 of 21 checks passed, 1 failed · 393 files scanned · 1458ms" {
		t.Fatalf("failing: got %q", failing)
	}
}
