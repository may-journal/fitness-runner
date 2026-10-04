package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRun(t *testing.T) {
	valid := map[string]string{
		manifestPath:  `{".":"1.2.3"}`,
		versionPath:   "1.2.3\n",
		changelogPath: "# Changelog\n\n## 1.2.3 (2026-10-04)\n\n- Add release correlation.\n\n## Changes\n",
	}
	cases := []struct {
		name       string
		files      map[string]string
		changed    string
		ok         bool
		checked    int
		wantErrors []string
	}{
		{name: "unrelated repository passes", ok: true},
		{name: "empty bootstrap manifest passes", files: map[string]string{manifestPath: "{}"}, ok: true, checked: 1},
		{name: "correlated release passes", files: valid, changed: manifestPath + "\n" + changelogPath, ok: true, checked: 3},
		{name: "asterisk release item passes", files: merge(valid, map[string]string{changelogPath: "## [1.2.3] (2026-10-04)\n\n* Fix publishing.\n"}), ok: true, checked: 3},
		{name: "manifest only change fails", files: valid, changed: manifestPath, checked: 3,
			wantErrors: []string{manifestPath + " changed without " + changelogPath + "; add the release section reviewers will approve"}},
		{name: "missing release section fails", files: merge(valid, map[string]string{changelogPath: "# Changelog\n\n## Changes\n\n### 2026.10.04.1200\n\n- Work.\n"}), checked: 3,
			wantErrors: []string{changelogPath + " must contain a `## 1.2.3` release section"}},
		{name: "empty release section fails", files: merge(valid, map[string]string{changelogPath: "## 1.2.3\n\n### Features\n\nText only.\n"}), checked: 3,
			wantErrors: []string{changelogPath + " release section 1.2.3 must contain at least one list item describing what ships"}},
		{name: "duplicate release section fails", files: merge(valid, map[string]string{changelogPath: "## 1.2.3\n- One.\n## 1.2.3 (today)\n- Two.\n"}), checked: 3,
			wantErrors: []string{changelogPath + " contains duplicate release sections for 1.2.3"}},
		{name: "version disagreement fails", files: merge(valid, map[string]string{versionPath: "1.2.4"}), checked: 3,
			wantErrors: []string{versionPath + ` version "1.2.4" must match ` + manifestPath + ` version "1.2.3"`}},
		{name: "malformed manifest fails", files: map[string]string{manifestPath: "{"}, checked: 1,
			wantErrors: []string{manifestPath + " is invalid JSON"}},
		{name: "null manifest fails", files: map[string]string{manifestPath: "null"}, checked: 1,
			wantErrors: []string{manifestPath + " must be a JSON object"}},
		{name: "manifest without root fails", files: map[string]string{manifestPath: `{"go":"1.2.3"}`}, checked: 1,
			wantErrors: []string{manifestPath + ` must contain a root "." release version`}},
		{name: "invalid semver fails", files: map[string]string{manifestPath: `{".":"v1.2.3"}`}, checked: 1,
			wantErrors: []string{manifestPath + ` root version must be numeric major.minor.patch (got "v1.2.3")`}},
		{name: "missing version file fails", files: map[string]string{manifestPath: `{".":"1.2.3"}`, changelogPath: valid[changelogPath]}, checked: 3,
			wantErrors: []string{versionPath + " is missing; it must match release 1.2.3"}},
		{name: "missing changelog fails", files: map[string]string{manifestPath: `{".":"1.2.3"}`, versionPath: "1.2.3"}, checked: 3,
			wantErrors: []string{changelogPath + " is missing; add a release section for 1.2.3"}},
		{name: "missing correlated files reports both", files: map[string]string{manifestPath: `{".":"1.2.3"}`}, checked: 3,
			wantErrors: []string{versionPath + " is missing; it must match release 1.2.3", changelogPath + " is missing; add a release section for 1.2.3"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, tc.files)
			t.Setenv("FITNESS_CHANGED_FILES", tc.changed)
			got, err := run(root, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got.Ok != tc.ok || got.FilesChecked != tc.checked || !slices.Equal(got.Errors, tc.wantErrors) {
				t.Fatalf("result = %+v, want ok=%v checked=%d errors=%q", got, tc.ok, tc.checked, tc.wantErrors)
			}
		})
	}
}

func TestIsAvailable(t *testing.T) {
	root := t.TempDir()
	available, err := isAvailable(root)
	if err != nil || available {
		t.Fatalf("absent manifest: available=%v err=%v", available, err)
	}
	writeFiles(t, root, map[string]string{manifestPath: "{}"})
	available, err = isAvailable(root)
	if err != nil || !available {
		t.Fatalf("present manifest: available=%v err=%v", available, err)
	}
}

func merge(base, override map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(override))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range override {
		out[key] = value
	}
	return out
}

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
