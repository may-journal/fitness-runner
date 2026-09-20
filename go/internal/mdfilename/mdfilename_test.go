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
	writeFiles(t, dir, "releaseNotes.md", "README.md", "api-design.md")
	res := Run(dir, Camel)
	if res.Ok {
		t.Fatalf("expected failure, got %+v", res)
	}
	if res.FilesChecked != 3 {
		t.Fatalf("filesChecked = %d, want 3", res.FilesChecked)
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

func TestRunSkipsPrunedDirectories(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "api-design.md", "node_modules/badName.md", "dist/AlsoBad.md")
	res := Run(dir, Kebab)
	if !res.Ok || res.FilesChecked != 1 {
		t.Fatalf("expected pass over 1 file with skip dirs pruned, got %+v", res)
	}
}

func TestRunEmptyRepoPasses(t *testing.T) {
	res := Run(t.TempDir(), Kebab)
	if !res.Ok || res.FilesChecked != 0 || len(res.Errors) != 0 {
		t.Fatalf("expected pass over 0 files, got %+v", res)
	}
}
