package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJudge(t *testing.T) {
	cases := []struct {
		name      string
		required  string
		current   string
		ok        bool
		wantError string
	}{
		{"equal major passes", "24", "v24.0.0", true, ""},
		{"greater major passes", "20", "v24.1.2", true, ""},
		{"lower major fails", "24", "v20.11.0", false,
			"Node v20.11.0 does not satisfy .nvmrc (requires 24.x). Run: nvm use"},
		{"v-prefixed requirement", "v24", "v24.5.0", true, ""},
		{"trailing minor in requirement", "24.1.0", "v24.0.0", true, ""},
		{"unparsable requirement reports NaN", "latest", "v24.0.0", false,
			"Node v24.0.0 does not satisfy .nvmrc (requires NaN.x). Run: nvm use"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := judge(tc.required, tc.current)
			if res.Ok != tc.ok {
				t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, tc.ok, res.Errors)
			}
			if res.FilesChecked != 1 {
				t.Fatalf("filesChecked = %d, want 1", res.FilesChecked)
			}
			if !tc.ok && res.Errors[0] != tc.wantError {
				t.Fatalf("error = %q, want %q", res.Errors[0], tc.wantError)
			}
		})
	}
}

func TestRunMissingNvmrc(t *testing.T) {
	dir := t.TempDir()
	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || res.Errors[0] != "missing .nvmrc" || res.FilesChecked != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestRunAgainstRealNvmrc(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".nvmrc"), []byte("1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	// any installed node satisfies a required major of 1
	if !res.Ok {
		t.Fatalf("expected pass against .nvmrc=1, got %+v", res)
	}
}
