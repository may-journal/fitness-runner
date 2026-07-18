package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// write creates rel under root with content, making parent dirs as needed.
func write(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindDisallowedEmphasis(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []hit
	}{
		{"plain text", "# Hi\n\nNo emphasis.", nil},
		{"detects **bold**", "x **y** z",
			[]hit{{kind: "**bold**", match: "**y**"}}},
		{"detects __bold__", "a __b__ c",
			[]hit{{kind: "__bold__", match: "__b__"}}},
		{"detects *italic* but not **", "*italic* and **bold**",
			[]hit{{kind: "**bold**", match: "**bold**"}, {kind: "*italic*", match: "*italic*"}}},
		{"detects _italic_", "_italic_ word",
			[]hit{{kind: "_italic_", match: "_italic_"}}},
		{"list markers not flagged", "* item one\n* item two\n* item three", nil},
		{"link blocks not flagged", "Link [_dev/path/to_resource_](path).", nil},
		{"inline and fenced code exempt", "Text with `**code**` and ```\n**block**\n``` and _real_ emphasis.",
			[]hit{{kind: "_italic_", match: "_real_"}}},
		{"lookbehind skip finds later italic", "**a*b*",
			[]hit{{kind: "*italic*", match: "*b*"}}},
		{"underscore italic may span lines", "_a\nb_",
			[]hit{{kind: "_italic_", match: "_a\nb_"}}},
		{"triple asterisks are one bold hit", "***shout***",
			[]hit{{kind: "**bold**", match: "**shout**"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := findDisallowedEmphasis(tc.content)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestValidateFileMessages(t *testing.T) {
	got := validateFile("e.md", "**B** and *i* and __u__.")
	want := []string{
		`e.md: disallowed **bold** (use only when explicitly required): "**B**"`,
		`e.md: disallowed __bold__ (use only when explicitly required): "__u__"`,
		`e.md: disallowed *italic* (use only when explicitly required): "*i*"`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestJSONStringifyQuoting(t *testing.T) {
	cases := []struct{ in, want string }{
		{"**world**", `"**world**"`},
		{`**say "hi"**`, `"**say \"hi\"**"`},
		{"_a\nb_", `"_a\nb_"`},
		{"**<b> & tab\t**", "\"**<b> & tab\\t**\""},
		{"_line\u2028sep_", `"_line\u2028sep_"`},
	}
	for _, tc := range cases {
		if got := jsonStringify(tc.in); got != tc.want {
			t.Errorf("jsonStringify(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestRun(t *testing.T) {
	cases := []struct {
		name       string
		files      map[string]string
		ok         bool
		wantErrors []string
		wantFiles  int
	}{
		{"no markdown files", map[string]string{}, true, nil, 0},
		{"clean markdown", map[string]string{
			"doc.md": "# Title\n\nPlain text and code `*not*`.\n",
		}, true, nil, 1},
		{"bold asterisk fails verbatim", map[string]string{
			"a.md": "Hello **world** here.",
		}, false, []string{
			`a.md: disallowed **bold** (use only when explicitly required): "**world**"`,
		}, 1},
		{"italic underscore fails verbatim", map[string]string{
			"d.md": "And _italic_ here.",
		}, false, []string{
			`d.md: disallowed _italic_ (use only when explicitly required): "_italic_"`,
		}, 1},
		{"root CHANGELOG.md exempt, nested one is not", map[string]string{
			"CHANGELOG.md":      "## 1.0.0 — **bold** allowed here",
			"docs/CHANGELOG.md": "Nested **bold**.",
		}, false, []string{
			`docs/CHANGELOG.md: disallowed **bold** (use only when explicitly required): "**bold**"`,
		}, 1},
		{"skip dirs pruned via cspell", map[string]string{
			"cspell.json":             `{"ignorePaths":["node_modules","dist","coverage",".git","githooks"]}`,
			"ok.md":                   "Plain.",
			"node_modules/pkg.md":     "**bold**",
			"dist/generated/notes.md": "_italic_",
		}, true, nil, 1},
		{"multiple files sorted, errors in file order", map[string]string{
			"b.md":     "Say __hello__.",
			"a/one.md": "This is *italic* text.",
		}, false, []string{
			`a/one.md: disallowed *italic* (use only when explicitly required): "*italic*"`,
			`b.md: disallowed __bold__ (use only when explicitly required): "__hello__"`,
		}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for rel, content := range tc.files {
				write(t, root, rel, content)
			}
			res, err := run(root, nil)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ok != tc.ok {
				t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, tc.ok, res.Errors)
			}
			if res.FilesChecked != tc.wantFiles {
				t.Fatalf("filesChecked = %d, want %d", res.FilesChecked, tc.wantFiles)
			}
			if tc.wantErrors != nil && !reflect.DeepEqual(res.Errors, tc.wantErrors) {
				t.Fatalf("errors = %v, want %v", res.Errors, tc.wantErrors)
			}
			if tc.wantErrors == nil && len(res.Errors) != 0 {
				t.Fatalf("expected no errors, got %v", res.Errors)
			}
		})
	}
}
