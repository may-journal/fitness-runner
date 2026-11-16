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
// checks declared fields, and yaml-files is new in a later index.
var fields0173 = map[string]map[string]string{
	"md-files":    {"files": `\.md$`},
	"go-files":    {"files": `\.go$`, "exclude": `(^|/)vendor/`},
	"swift-files": {"files": `\.swift$`},
	"no-fields":   {},
	"yaml-files":  {"files": `\.ya?ml$`},
}

// firstIndex0173 is the version of each check the first index publishes.
var firstIndex0173 = map[string]string{"md-files": "1.0.0", "go-files": "1.0.0", "swift-files": "1.0.0", "no-fields": "1.0.0"}

// pins0173 pins every fake check at its major.
const pins0173 = `{"checkVersions": {"md-files": "v1", "go-files": "v1", "swift-files": "v1", "no-fields": "v1"}}`

// checks0173 is the --checks flag that runs every fake check.
const checks0173 = "--checks=md-files,go-files,swift-files,no-fields"

// releases0173 is the fake checks' server, which records which ones a run
// downloads, and the private cache whose index names them.
type releases0173 struct {
	url, cache string
	archives   map[string][]byte
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
	r := &releases0173{cache: t.TempDir(), archives: map[string][]byte{}}
	for name := range fields0173 {
		r.archives[name] = archive0173(t, name)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		name := strings.TrimSuffix(strings.TrimPrefix(req.URL.Path, "/"), ".tar")
		r.mu.Lock()
		r.fetched = append(r.fetched, name)
		r.mu.Unlock()
		body := r.archives[name]
		if slices.Contains(tampered, name) {
			body = append(slices.Clone(body), 0)
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	r.url = srv.URL
	r.publish(t, firstIndex0173, time.Now())
	return r
}

// publish records in r's cache a release index read at checked. It names
// each check in versions at its version, the newest of major 1.
func (r *releases0173) publish(t *testing.T, versions map[string]string, checked time.Time) {
	t.Helper()
	checks := map[string]any{}
	for name, number := range versions {
		version := map[string]any{
			"tag":       "fitness-check-" + name + "-v" + number,
			"artifacts": map[string]any{runtime.GOOS + "-" + runtime.GOARCH: map[string]string{"url": r.url + "/" + name + ".tar", "sha256": fmt.Sprintf("%x", sha256.Sum256(r.archives[name]))}},
		}
		for key, value := range fields0173[name] {
			version[key] = value
		}
		checks[name] = map[string]any{"majors": map[string]string{"1": number}, "versions": map[string]any{number: version}}
	}
	record, err := json.Marshal(map[string]any{"value": map[string]any{"format": 1, "checks": checks}, "checked": checked.Format(time.RFC3339)})
	mustDo(t, err)
	mustDo(t, os.WriteFile(filepath.Join(r.cache, "check-index.json"), record, 0o600))
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

// hooked0173 is the environment the shared hooks give fitness-install, with
// a private cache that holds the installed engine and r's index.
func hooked0173(t *testing.T, r *releases0173) []string {
	t.Helper()
	cache := fiPrivateCache(t)
	mustDo(t, os.Rename(filepath.Join(r.cache, "check-index.json"), filepath.Join(strings.TrimPrefix(cache[0], "FITNESS_CACHE_DIR="), "check-index.json")))
	return append(cache, "FITNESS_CHECKS_DIR=", "FITNESS_WITH_CHECKS=pinned")
}

func Test0173_4(t *testing.T) {
	t.Parallel()
	r := serve0173(t)
	out, code := fiRun(t, staged0173(t, map[string]string{"notes/plan.md": "# Plan\n"}), hooked0173(t, r), "--", "--policy=external", checks0173)
	sees(t, out, code, 1, "md-files ran", "Skipped (no matching files): go-files, swift-files")
	if installedIn0164.MatchString(out) {
		t.Errorf("fitness-install must install no check up front:\n%s", out)
	}
	fetchedOnly(t, r, out, "md-files", "no-fields")
}

func Test0173_5(t *testing.T) {
	t.Parallel()
	r := serve0173(t)
	out, code := fiRun(t, staged0173(t, nil), hooked0173(t, r), "--", "config")
	sees(t, out, code, 0, "installed 4 checks in ")
	fetchedOnly(t, r, out, "go-files", "md-files", "no-fields", "swift-files")
}

// cached0173 fails unless r's cache holds check name's program at version.
func cached0173(t *testing.T, r *releases0173, name, version string) {
	t.Helper()
	tag := "fitness-check-" + name + "-v" + version
	if _, err := os.Stat(filepath.Join(r.cache, "checks", tag+"-"+runtime.GOOS+"-"+runtime.GOARCH+".tar", "fitness-check-"+name)); err != nil {
		t.Errorf("the cache must hold %s %s: %v", name, version, err)
	}
}

func Test0173_6(t *testing.T) {
	t.Parallel()
	r := serve0173(t)
	bundle := bundle0162(t)
	repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"checkVersions": {"*": "v1"}}`})
	write(t, repo, map[string]string{"notes/plan.md": "# Plan\n"})
	git(t, repo, "add", "-A")
	env := []string{"FITNESS_CACHE_DIR=" + r.cache}
	out, code := run0162(t, bundle, repo, env, "--policy=external", "--checks=md-files")
	sees(t, out, code, 1, "md-files ran")
	cached0173(t, r, "md-files", "1.0.0")

	// Ten minutes on, the recorded index is stale, so the run asks GitHub again.
	r.publish(t, firstIndex0173, time.Now().Add(-11*time.Minute))
	silent := append(slices.Clone(env), "HTTPS_PROXY="+silentListener(t), "NO_PROXY=", "no_proxy=")
	out, code = run0162(t, bundle, repo, silent, "--policy=external", "--checks=md-files")
	sees(t, out, code, 1, "asking github.com for the check release index", "GitHub did not answer the check release index")
	fetchedOnly(t, r, out, "md-files")

	// The index lives on GitHub, so the test records the answer that read gets.
	r.publish(t, map[string]string{"md-files": "1.1.0", "go-files": "1.0.0", "swift-files": "1.0.0", "no-fields": "1.0.0", "yaml-files": "1.0.0"}, time.Now())
	out, code = run0162(t, bundle, repo, env, "--policy=external", "--checks=md-files,yaml-files")
	sees(t, out, code, 1, "md-files ran", "Skipped (no matching files): yaml-files")
	cached0173(t, r, "md-files", "1.1.0")
	fetchedOnly(t, r, out, "md-files", "md-files")

	write(t, repo, map[string]string{"config/app.yaml": "name: app\n"})
	git(t, repo, "add", "-A")
	out, code = run0162(t, bundle, repo, env, "--policy=external", "--checks=md-files,yaml-files")
	sees(t, out, code, 1, "md-files ran", "yaml-files ran")
	fetchedOnly(t, r, out, "md-files", "md-files", "yaml-files")
}
