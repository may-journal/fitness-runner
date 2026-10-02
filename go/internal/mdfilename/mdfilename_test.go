package mdfilename

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		name       string
		relPath    string
		convention Convention
		want       string
	}{
		{"kebab accepts hyphenated name", "api-design.md", Kebab, ""},
		{"kebab accepts plain alphanumeric", "adr001.md", Kebab, ""},
		{"kebab rejects camelCase", "releaseNotes.md", Kebab,
			"releaseNotes.md: filename must be kebab-case"},
		{"kebab rejects double hyphen", "api--design.md", Kebab,
			"api--design.md: filename must be kebab-case"},
		{"kebab rejects underscore", "api_design.md", Kebab,
			"api_design.md: filename must be kebab-case"},
		{"kebab rejects mixed case", "Todo.md", Kebab,
			"Todo.md: filename must be kebab-case"},
		{"camel accepts camelCase", "releaseNotes.md", Camel, ""},
		{"camel accepts plain alphanumeric", "adr001.md", Camel, ""},
		{"camel rejects hyphenated name", "api-design.md", Camel,
			"api-design.md: filename must be camelCase"},
		{"camel rejects uppercase start", "ReleaseNotes.md", Camel,
			"ReleaseNotes.md: filename must be camelCase"},
		{"caps README exempt from kebab", "README.md", Kebab, ""},
		{"caps AGENTS exempt from kebab", "AGENTS.md", Kebab, ""},
		{"caps AGENTS exempt from camel", "AGENTS.md", Camel, ""},
		{"caps underscore doc exempt", "CODE_OF_CONDUCT.md", Kebab, ""},
		{"caps doc exempt in nested path", "docs/README.md", Kebab, ""},
		{"caps AGENTS exempt in nested path", "docs/AGENTS.md", Kebab, ""},
		{"nested path appears in error", "docs/releaseNotes.md", Kebab,
			"docs/releaseNotes.md: filename must be kebab-case"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Validate(tc.relPath, tc.convention); got != tc.want {
				t.Fatalf("Validate(%q) = %q, want %q", tc.relPath, got, tc.want)
			}
		})
	}
}

// writeFiles creates each relative path under dir with placeholder content.
func writeFiles(t *testing.T, dir string, relPaths ...string) {
	t.Helper()
	for _, rel := range relPaths {
		abs := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("# doc\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRunKebabFlavor(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "api-design.md", "README.md", "releaseNotes.md")
	res := Run(dir, Kebab)
	if res.Ok {
		t.Fatalf("expected failure, got %+v", res)
	}
	if res.FilesChecked != 3 {
		t.Fatalf("filesChecked = %d, want 3", res.FilesChecked)
	}
	want := "releaseNotes.md: filename must be kebab-case"
	if len(res.Errors) != 1 || res.Errors[0] != want {
		t.Fatalf("errors = %v, want [%q]", res.Errors, want)
	}
}

func TestRunCamelFlavor(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "releaseNotes.md", "installGuide.md", "README.md", "api-design.md")
	res := Run(dir, Camel)
	if res.Ok {
		t.Fatalf("expected failure, got %+v", res)
	}
	if res.FilesChecked != 4 {
		t.Fatalf("filesChecked = %d, want 4", res.FilesChecked)
	}
	want := "api-design.md: filename must be camelCase"
	if len(res.Errors) != 1 || res.Errors[0] != want {
		t.Fatalf("errors = %v, want [%q]", res.Errors, want)
	}
}

func TestRunAllConformingPasses(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "api-design.md", "docs/getting-started.md", "SECURITY.md")
	res := Run(dir, Kebab)
	if !res.Ok || len(res.Errors) != 0 || res.FilesChecked != 3 {
		t.Fatalf("expected clean pass over 3 files, got %+v", res)
	}
}

func TestRunSkipsNoDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "api-design.md", "node_modules/badName.md", "dist/AlsoBad.md")
	res := Run(dir, Kebab)
	if res.Ok || res.FilesChecked != 3 || len(res.Errors) != 2 {
		t.Fatalf("expected node_modules and dist judged too, got %+v", res)
	}
}

func TestRunEmptyRepoPasses(t *testing.T) {
	res := Run(t.TempDir(), Kebab)
	if !res.Ok || res.FilesChecked != 0 || len(res.Errors) != 0 {
		t.Fatalf("expected pass over 0 files, got %+v", res)
	}
}

func TestDetect(t *testing.T) {
	cases := []struct {
		name  string
		files []string
		want  Convention
	}{
		{"no markdown defaults to kebab", nil, Kebab},
		{"only neutral and exempt names default to kebab", []string{"README.md", "adr001.md", "notes.md"}, Kebab},
		{"hyphenated majority is kebab", []string{"api-design.md", "release-notes.md", "installGuide.md"}, Kebab},
		{"capitalized majority is camel", []string{"releaseNotes.md", "docs/installGuide.md", "api-design.md"}, Camel},
		{"a tie goes to kebab", []string{"releaseNotes.md", "api-design.md"}, Kebab},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Detect(tc.files); got.Label != tc.want.Label {
				t.Errorf("Detect = %s, want %s", got.Label, tc.want.Label)
			}
		})
	}
}

func TestRunOtherFlavorPassesClean(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "releaseNotes.md", "installGuide.md", "api-design.md")
	if res := Run(dir, Kebab); !res.Ok || res.FilesChecked != 0 {
		t.Fatalf("kebab flavor in a camelCase repo = %+v, want a clean pass", res)
	}
}

func TestRunScopedJudgesChangedFiles(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "api-design.md", "old-notes.md", "releaseNotes.md", "badName.md")
	t.Setenv("FITNESS_CHANGED_FILES", "badName.md")
	res := Run(dir, Kebab)
	if res.FilesChecked != 1 {
		t.Fatalf("filesChecked = %d, want 1", res.FilesChecked)
	}
	want := "badName.md: filename must be kebab-case"
	if len(res.Errors) != 1 || res.Errors[0] != want {
		t.Fatalf("errors = %v, want [%q]", res.Errors, want)
	}
}
