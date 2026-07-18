package render

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func plain() Palette { return Palette{} }

func TestTableGeometryAt80(t *testing.T) {
	rows := []Row{
		{Name: "read-repo-first", Ok: true, FilesChecked: 0, Ms: 7},
		{Name: "markdown-filename-kebab-case", Ok: false, FilesChecked: 49, Ms: 6,
			Errors: []string{"some/path.md: filename must be kebab-case"}},
	}
	out := Table(rows, 80, plain())
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if got := utf8.RuneCountInString(line); got != 80 {
			t.Errorf("line %d width = %d, want 80: %q", i, got, line)
		}
	}
	if !strings.Contains(out, "│ markdown-filename-kebab-c… │") {
		t.Error("long name must truncate with ellipsis at 26 chars")
	}
	if !strings.Contains(out, "[markdown-filename-kebab-case] Please fix these items:") {
		t.Error("error block header missing")
	}
	if !strings.Contains(out, "  ✖ some/path.md: filename must be kebab-case") {
		t.Error("error bullet missing")
	}
	if !strings.Contains(out, "│ read-repo-first            │ passed   │ 0      │ 7ms") {
		t.Error("row cells misaligned")
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
	for i, line := range strings.Split(out, "\n") {
		if got := utf8.RuneCountInString(line); got != 80 {
			t.Fatalf("wrapped line %d width = %d: %q", i, got, line)
		}
	}
	if !strings.Contains(out, "│   ✖ word") {
		t.Error("first wrapped line must keep the bullet indent")
	}
}

func TestTotalLine(t *testing.T) {
	got := TotalLine(21, 0, 393, 1458, plain())
	if got != "Total: 21 succeeded, 0 failed, 393 files in 1458ms" {
		t.Fatalf("got %q", got)
	}
}
