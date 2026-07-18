package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

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

// reply is one canned mock-registry response.
type reply struct {
	status int
	body   string
}

// latestBody is the abbreviated-metadata body naming version as latest.
func latestBody(version string) string {
	return `{"dist-tags":{"latest":"` + version + `"}}`
}

// pkgBody is an installed node_modules package.json at version.
func pkgBody(version string) string {
	return `{"name":"x","version":"` + version + `"}`
}

// newRegistry serves canned replies keyed by package name (unknown names
// get 404) and asserts every request carries the abbreviated-metadata
// Accept header.
func newRegistry(t *testing.T, replies map[string]reply) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if accept := r.Header.Get("Accept"); accept != "application/vnd.npm.install-v1+json" {
			t.Errorf("Accept = %q, want abbreviated metadata", accept)
		}
		res, ok := replies[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(res.status)
		_, _ = io.WriteString(w, res.body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRun(t *testing.T) {
	cases := []struct {
		name         string
		files        map[string]string
		replies      map[string]reply
		offline      bool
		ok           bool
		wantErrors   []string
		filesChecked int
	}{
		{
			name: "passes when everything is current",
			files: map[string]string{
				"package.json":                    `{"dependencies":{"chalk":"^5.3.0"}}`,
				"node_modules/chalk/package.json": pkgBody("5.3.0"),
			},
			replies:      map[string]reply{"chalk": {200, latestBody("5.3.0")}},
			ok:           true,
			filesChecked: 1,
		},
		{
			name: "fails and lists deps behind latest, sorted after the lead",
			files: map[string]string{
				"package.json":                    `{"dependencies":{"zod":"^3.0.0","chalk":"^4.1.2"}}`,
				"node_modules/chalk/package.json": pkgBody("4.1.2"),
				"node_modules/zod/package.json":   pkgBody("3.0.0"),
			},
			replies: map[string]reply{
				"chalk": {200, latestBody("5.3.0")},
				"zod":   {200, latestBody("3.23.0")},
			},
			ok: false,
			wantErrors: []string{
				"Dependencies behind their latest published version — update them (npm install <pkg>@latest) or pin intentionally:",
				"chalk: 4.1.2 → 5.3.0",
				"zod: 3.0.0 → 3.23.0",
			},
			filesChecked: 1,
		},
		{
			name: "reports a declared but not installed dependency as missing",
			files: map[string]string{
				"package.json": `{"dependencies":{"foo":"^2.0.0"}}`,
			},
			replies:      map[string]reply{"foo": {200, latestBody("2.0.0")}},
			ok:           false,
			wantErrors:   []string{lead, "foo: missing → 2.0.0"},
			filesChecked: 1,
		},
		{
			name: "skips internal @mayjournal packages",
			files: map[string]string{
				"package.json": `{"dependencies":{"@mayjournal/fitness-shared":"*"}}`,
			},
			replies:      map[string]reply{"@mayjournal/fitness-shared": {200, latestBody("9.9.9")}},
			ok:           true,
			filesChecked: 1,
		},
		{
			name: "devDependencies and peerDependencies are enumerated",
			files: map[string]string{
				"package.json":                  `{"devDependencies":{"foo":"^1.0.0"},"peerDependencies":{"bar":"^1.0.0"}}`,
				"node_modules/foo/package.json": pkgBody("1.0.0"),
				"node_modules/bar/package.json": pkgBody("1.0.0"),
			},
			replies: map[string]reply{
				"foo": {200, latestBody("2.0.0")},
				"bar": {200, latestBody("1.5.0")},
			},
			ok:           false,
			wantErrors:   []string{lead, "bar: 1.0.0 → 1.5.0", "foo: 1.0.0 → 2.0.0"},
			filesChecked: 1,
		},
		{
			name: "scoped dependency resolves and reports",
			files: map[string]string{
				"package.json":                          `{"devDependencies":{"@types/node":"^25.0.0"}}`,
				"node_modules/@types/node/package.json": pkgBody("25.3.0"),
			},
			replies:      map[string]reply{"@types/node": {200, latestBody("26.1.0")}},
			ok:           false,
			wantErrors:   []string{lead, "@types/node: 25.3.0 → 26.1.0"},
			filesChecked: 1,
		},
		{
			name: "garbage registry response contributes nothing",
			files: map[string]string{
				"package.json":                    `{"dependencies":{"chalk":"^4.1.2"}}`,
				"node_modules/chalk/package.json": pkgBody("4.1.2"),
			},
			replies:      map[string]reply{"chalk": {200, "npm error code ENOTFOUND"}},
			ok:           true,
			filesChecked: 1,
		},
		{
			name: "non-200 response contributes nothing",
			files: map[string]string{
				"package.json":                    `{"dependencies":{"chalk":"^4.1.2"}}`,
				"node_modules/chalk/package.json": pkgBody("4.1.2"),
			},
			replies:      map[string]reply{"chalk": {500, latestBody("5.3.0")}},
			ok:           true,
			filesChecked: 1,
		},
		{
			name: "response without dist-tags contributes nothing",
			files: map[string]string{
				"package.json":                    `{"dependencies":{"chalk":"^4.1.2"}}`,
				"node_modules/chalk/package.json": pkgBody("4.1.2"),
			},
			replies:      map[string]reply{"chalk": {200, "{}"}},
			ok:           true,
			filesChecked: 1,
		},
		{
			name: "unreachable registry degrades to a pass",
			files: map[string]string{
				"package.json":                    `{"dependencies":{"chalk":"^4.1.2"}}`,
				"node_modules/chalk/package.json": pkgBody("4.1.2"),
			},
			offline:      true,
			ok:           true,
			filesChecked: 1,
		},
		{
			name: "workspace dependencies are enumerated via root globs",
			files: map[string]string{
				"package.json":                  `{"workspaces":["packages/*"]}`,
				"packages/a/package.json":       `{"dependencies":{"zod":"^3.0.0"}}`,
				"node_modules/zod/package.json": pkgBody("3.0.0"),
			},
			replies:      map[string]reply{"zod": {200, latestBody("3.23.0")}},
			ok:           false,
			wantErrors:   []string{lead, "zod: 3.0.0 → 3.23.0"},
			filesChecked: 2,
		},
		{
			name: "workspace node_modules wins over the hoisted root install",
			files: map[string]string{
				"package.json":                             `{"workspaces":["packages/*"]}`,
				"packages/a/package.json":                  `{"dependencies":{"zod":"^3.0.0"}}`,
				"packages/a/node_modules/zod/package.json": pkgBody("3.23.0"),
				"node_modules/zod/package.json":            pkgBody("3.0.0"),
			},
			replies:      map[string]reply{"zod": {200, latestBody("3.23.0")}},
			ok:           true,
			filesChecked: 2,
		},
		{
			name: "identical lines across workspaces collapse to one",
			files: map[string]string{
				"package.json":                     `{"workspaces":["packages/*"]}`,
				"packages/a/package.json":          `{"devDependencies":{"vitest":"^3.0.0"}}`,
				"packages/b/package.json":          `{"devDependencies":{"vitest":"^3.0.0"}}`,
				"node_modules/vitest/package.json": pkgBody("3.0.0"),
			},
			replies:      map[string]reply{"vitest": {200, latestBody("4.0.18")}},
			ok:           false,
			wantErrors:   []string{lead, "vitest: 3.0.0 → 4.0.18"},
			filesChecked: 3,
		},
		{
			name: "different installed versions across workspaces both report",
			files: map[string]string{
				"package.json":                                `{"workspaces":["packages/*"]}`,
				"packages/a/package.json":                     `{"devDependencies":{"vitest":"^2.0.0"}}`,
				"packages/a/node_modules/vitest/package.json": pkgBody("2.0.0"),
				"packages/b/package.json":                     `{"devDependencies":{"vitest":"^3.0.0"}}`,
				"packages/b/node_modules/vitest/package.json": pkgBody("3.0.0"),
			},
			replies:      map[string]reply{"vitest": {200, latestBody("4.0.18")}},
			ok:           false,
			wantErrors:   []string{lead, "vitest: 2.0.0 → 4.0.18", "vitest: 3.0.0 → 4.0.18"},
			filesChecked: 3,
		},
		{
			name: "object-form workspaces field is honored",
			files: map[string]string{
				"package.json":            `{"workspaces":{"packages":["packages/a"]}}`,
				"packages/a/package.json": `{"dependencies":{"foo":"^1.0.0"}}`,
			},
			replies:      map[string]reply{"foo": {200, latestBody("2.0.0")}},
			ok:           false,
			wantErrors:   []string{lead, "foo: missing → 2.0.0"},
			filesChecked: 2,
		},
		{
			name:         "no root package.json passes with the walk count",
			files:        map[string]string{"docs/readme.md": "hi"},
			ok:           true,
			filesChecked: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for rel, content := range tc.files {
				write(t, root, rel, content)
			}
			srv := newRegistry(t, tc.replies)
			if tc.offline {
				srv.Close()
			}
			write(t, root, ".npmrc", "registry="+srv.URL+"\n")
			res, err := run(root, nil)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ok != tc.ok {
				t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, tc.ok, res.Errors)
			}
			if res.FilesChecked != tc.filesChecked {
				t.Fatalf("filesChecked = %d, want %d", res.FilesChecked, tc.filesChecked)
			}
			if tc.ok {
				if len(res.Errors) != 0 {
					t.Fatalf("errors = %v, want none", res.Errors)
				}
				return
			}
			if !reflect.DeepEqual(res.Errors, tc.wantErrors) {
				t.Fatalf("errors = %#v, want %#v", res.Errors, tc.wantErrors)
			}
		})
	}
}

func TestEntryLine(t *testing.T) {
	cases := []struct {
		name      string
		installed string
		latest    string
		want      string
	}{
		{"unknown latest contributes nothing", "4.1.2", "", ""},
		{"already at latest contributes nothing", "5.3.0", "5.3.0", ""},
		{"behind latest formats current → latest", "4.1.2", "5.3.0", "chalk: 4.1.2 → 5.3.0"},
		{"not installed reports missing", "", "5.3.0", "chalk: missing → 5.3.0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := entryLine("chalk", tc.installed, tc.latest); got != tc.want {
				t.Fatalf("entryLine = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNpmrcRegistry(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"plain assignment", "registry=https://r.example.com\n", "https://r.example.com"},
		{"trailing slash trimmed", "registry=https://r.example.com/\n", "https://r.example.com"},
		{"spaces around equals", "registry = https://r.example.com\n", "https://r.example.com"},
		{"comments and other keys skipped",
			"# a comment\n; another\n//registry.npmjs.org/:_authToken=abc\nregistry=https://r.example.com\n",
			"https://r.example.com"},
		{"no registry key", "//registry.npmjs.org/:_authToken=abc\n", ""},
		{"empty value ignored", "registry=\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			write(t, root, ".npmrc", tc.content)
			if got := npmrcRegistry(filepath.Join(root, ".npmrc")); got != tc.want {
				t.Fatalf("npmrcRegistry = %q, want %q", got, tc.want)
			}
		})
	}
	t.Run("missing file", func(t *testing.T) {
		if got := npmrcRegistry(filepath.Join(t.TempDir(), ".npmrc")); got != "" {
			t.Fatalf("npmrcRegistry = %q, want empty", got)
		}
	})
}

func TestRegistryFrom(t *testing.T) {
	t.Run("root .npmrc wins over home", func(t *testing.T) {
		root, home := t.TempDir(), t.TempDir()
		t.Setenv("HOME", home)
		write(t, root, ".npmrc", "registry=https://root.example.com\n")
		write(t, home, ".npmrc", "registry=https://home.example.com\n")
		if got := registryFrom(root); got != "https://root.example.com" {
			t.Fatalf("registryFrom = %q", got)
		}
	})
	t.Run("falls back to home .npmrc", func(t *testing.T) {
		root, home := t.TempDir(), t.TempDir()
		t.Setenv("HOME", home)
		write(t, home, ".npmrc", "registry=https://home.example.com\n")
		if got := registryFrom(root); got != "https://home.example.com" {
			t.Fatalf("registryFrom = %q", got)
		}
	})
	t.Run("defaults to the public registry", func(t *testing.T) {
		root, home := t.TempDir(), t.TempDir()
		t.Setenv("HOME", home)
		if got := registryFrom(root); got != defaultRegistry {
			t.Fatalf("registryFrom = %q", got)
		}
	})
}
