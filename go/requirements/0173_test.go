// cspell:ignore Typeflag
package requirements

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// fields0173 are the index fields of each fake check. One is from before
// checks declared fields.
var fields0173 = map[string]map[string]string{
	"md-files":    {"files": `\.md$`},
	"go-files":    {"files": `\.go$`, "exclude": `(^|/)vendor/`},
	"swift-files": {"files": `\.swift$`},
	"no-fields":   {},
}

// pins0173 pins every fake check at its major.
const pins0173 = `{"checkVersions": {"md-files": "v1", "go-files": "v1", "swift-files": "v1", "no-fields": "v1"}}`

// checks0173 is the --checks flag that runs every fake check.
const checks0173 = "--checks=md-files,go-files,swift-files,no-fields"

// releases0173 is the fake checks' server, which records which ones a run
// downloads, and the private cache whose index names them.
type releases0173 struct {
	url, cache string
	mu         sync.Mutex
	fetched    []string
}

// downloaded lists the checks downloaded so far, in name order.
func (r *releases0173) downloaded() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Sorted(slices.Values(r.fetched))
}

// archive0173 is a check release archive holding the program of name.
func archive0173(t *testing.T, name string) []byte {
	t.Helper()
	program := []byte(script0172(name, fmt.Sprintf(`{"name":%q}`, name)))
	var buf bytes.Buffer
	w := tar.NewWriter(&buf)
	mustDo(t, w.WriteHeader(&tar.Header{Name: "fitness-check-" + name, Mode: 0o755, Size: int64(len(program)), Typeflag: tar.TypeReg}))
	_, err := w.Write(program)
	mustDo(t, err)
	mustDo(t, w.Close())
	return buf.Bytes()
}

// serve0173 serves the fake checks and seeds a fresh release index naming
// them, with their fields, into a private cache. A name in tampered serves
// bytes its SHA-256 does not match.
func serve0173(t *testing.T, tampered ...string) *releases0173 {
	t.Helper()
	r := &releases0173{cache: t.TempDir()}
	archives := map[string][]byte{}
	for name := range fields0173 {
		archives[name] = archive0173(t, name)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		name := strings.TrimSuffix(strings.TrimPrefix(req.URL.Path, "/"), ".tar")
		r.mu.Lock()
		r.fetched = append(r.fetched, name)
		r.mu.Unlock()
		body := archives[name]
		if slices.Contains(tampered, name) {
			body = append(slices.Clone(body), 0)
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	r.url = srv.URL
	checks := map[string]any{}
	for name, fields := range fields0173 {
		version := map[string]any{
			"tag":       "fitness-check-" + name + "-v1.0.0",
			"artifacts": map[string]any{runtime.GOOS + "-" + runtime.GOARCH: map[string]string{"url": srv.URL + "/" + name + ".tar", "sha256": fmt.Sprintf("%x", sha256.Sum256(archives[name]))}},
		}
		for key, value := range fields {
			version[key] = value
		}
		checks[name] = map[string]any{"majors": map[string]string{"1": "1.0.0"}, "versions": map[string]any{"1.0.0": version}}
	}
	record, err := json.Marshal(map[string]any{"value": map[string]any{"format": 1, "checks": checks}, "checked": time.Now().Format(time.RFC3339)})
	mustDo(t, err)
	mustDo(t, os.WriteFile(filepath.Join(r.cache, "check-index.json"), record, 0o600))
	return r
}

// staged0173 is a repo pinning every fake check, with files staged.
func staged0173(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": pins0173})
	write(t, repo, files)
	git(t, repo, "add", "-A")
	return repo
}

// fetchRun0173 runs the engine in repo with r's cache, as a clean machine
// with no checks handed down.
func fetchRun0173(t *testing.T, r *releases0173, repo string, args ...string) (string, int) {
	t.Helper()
	return run0162(t, bundle0162(t), repo, []string{"FITNESS_CACHE_DIR=" + r.cache}, append([]string{"--policy=external", checks0173}, args...)...)
}

// fetchedOnly fails unless r downloaded exactly want.
func fetchedOnly(t *testing.T, r *releases0173, out string, want ...string) {
	t.Helper()
	if got := r.downloaded(); !slices.Equal(got, slices.Sorted(slices.Values(want))) {
		t.Errorf("the run must download only %q, got %q:\n%s", want, got, out)
	}
}

func Test0173_1(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		staged  map[string]string
		reached string
		skipped string
	}{
		{map[string]string{"notes/plan.md": "# Plan\n"}, "md-files", "go-files, swift-files"},
		{map[string]string{"src/main.go": "package main\n"}, "go-files", "md-files, swift-files"},
		{map[string]string{"app/App.swift": "import Foundation\n"}, "swift-files", "md-files, go-files"},
	} {
		t.Run(c.reached, func(t *testing.T) {
			t.Parallel()
			r := serve0173(t)
			out, code := fetchRun0173(t, r, staged0173(t, c.staged))
			sees(t, out, code, 1, c.reached+" ran", "no-fields ran", "Skipped (no matching files): "+c.skipped)
			fetchedOnly(t, r, out, c.reached, "no-fields")
		})
	}
}

func Test0173_2(t *testing.T) {
	t.Parallel()
	r := serve0173(t)
	out, code := fetchRun0173(t, r, staged0173(t, map[string]string{"notes/plan.md": "# Plan\n"}), "--all")
	sees(t, out, code, 1, "md-files ran", "go-files ran", "swift-files ran", "no-fields ran")
	fetchedOnly(t, r, out, "go-files", "md-files", "no-fields", "swift-files")
}

func Test0173_3(t *testing.T) {
	t.Parallel()
	r := serve0173(t, "md-files")
	out, code := fetchRun0173(t, r, staged0173(t, map[string]string{"notes/plan.md": "# Plan\n"}))
	sees(t, out, code, 1, "check md-files applies to this run's files but could not be fetched", "checksum mismatch", "no-fields ran")
	absent(t, out, "md-files")
	if !strings.Contains(row(out, "md-files"), "info") {
		t.Errorf("a check that could not be fetched must be an info row, never a pass:\n%s", out)
	}
}

func Test0173_4(t *testing.T) {
	t.Parallel()
	r := serve0173(t)
	cache := fiPrivateCache(t)
	mustDo(t, os.Rename(filepath.Join(r.cache, "check-index.json"), filepath.Join(strings.TrimPrefix(cache[0], "FITNESS_CACHE_DIR="), "check-index.json")))
	env := append(cache, "FITNESS_CHECKS_DIR=", "FITNESS_WITH_CHECKS=pinned")
	out, code := fiRun(t, staged0173(t, map[string]string{"notes/plan.md": "# Plan\n"}), env, "--", "--policy=external", checks0173)
	sees(t, out, code, 1, "md-files ran", "Skipped (no matching files): go-files, swift-files")
	if installedIn0164.MatchString(out) {
		t.Errorf("fitness-install must install no check up front:\n%s", out)
	}
	fetchedOnly(t, r, out, "md-files", "no-fields")
}
