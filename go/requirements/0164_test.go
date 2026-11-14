package requirements

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

// installedIn0164 matches the line naming the folder of installed checks.
var installedIn0164 = regexp.MustCompile(`installed (\d+) checks in (\S+)`)

// tags0164 serves tags as the check release tag lookup. It returns the
// environment pointing the installer at it, with a private cache and no token.
func tags0164(t *testing.T, tags ...string) []string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refs := make([]string, len(tags))
		for i, tag := range tags {
			refs[i] = fmt.Sprintf(`{"ref":"refs/tags/%s"}`, tag)
		}
		fmt.Fprint(w, "["+strings.Join(refs, ",")+"]")
	}))
	t.Cleanup(srv.Close)
	return append(fiPrivateCache(t), "GITHUB_API_URL="+srv.URL, "GH_TOKEN=", "GITHUB_TOKEN=")
}

// pinned0164 is a repo whose checkVersions holds pins.
func pinned0164(t *testing.T, pins string) string {
	t.Helper()
	return example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"checkVersions": ` + pins + `}`})
}

func Test0164_1(t *testing.T) {
	t.Parallel()
	env := tags0164(t, "fitness-check-gofmt-v1.0.0", "fitness-check-gofmt-v0.9.0", "fitness-check-gofmt-extra-v1.9.0")
	repo := pinned0164(t, `{"markdown-links": "v1.0.0", "gofmt": "v1"}`)
	out, code := fiRun(t, repo, env, "--with-checks", "--install-only")
	sees(t, out, code, 0, "installed 2 checks in ")
	dir := installedIn0164.FindStringSubmatch(out)[2]
	entries, err := os.ReadDir(dir)
	mustDo(t, err)
	names := slices.Collect(func(yield func(string) bool) {
		for _, e := range entries {
			yield(e.Name())
		}
	})
	if !slices.Equal(names, []string{"fitness-check-gofmt", "fitness-check-markdown-links"}) {
		t.Fatalf("the folder must hold exactly the two pinned checks, got %q", names)
	}
	described, code := execute(t, func() *exec.Cmd { return exec.Command(filepath.Join(dir, "fitness-check-gofmt"), "--describe") })
	sees(t, described, code, 0, `"name":"gofmt"`, `"protocol":1`)
	again, code := fiRun(t, repo, env, "--with-checks", "--install-only")
	sees(t, again, code, 0, "installed 2 checks in "+dir)
	if strings.Contains(again, "downloading fitness-check-") || strings.Contains(again, "for the check release") {
		t.Errorf("a second install must reuse the cached checks and lookups:\n%s", again)
	}
}

func Test0164_2(t *testing.T) {
	t.Parallel()
	env := tags0164(t, "fitness-check-gofmt-v1.0.0")
	repo := pinned0164(t, `{"gofmt": "v2"}`)
	out, code := fiRun(t, repo, env, "--with-checks", "--install-only")
	sees(t, out, code, 0, "warning: no release of check gofmt matches v2; its built-in runs", "installed 0 checks in ")
	out, code = fiRun(t, repo, env, "--with-checks", "--", "--no-builtins", "gofmt")
	sees(t, out, code, 1, "no release of check gofmt matches v2")
}

func Test0164_3(t *testing.T) {
	t.Parallel()
	repo := pinned0164(t, `{"markdown-links": "v1.0.0"}`)
	commit(t, repo, map[string]string{"broken.md": "# Broken\n\nSee [the guide](missing.md).\n"}, "docs(broken): link a missing guide")
	env := tags0164(t)
	for _, run := range [][]string{{"--no-builtins"}, nil} {
		out, code := fiRun(t, repo, env, append(append([]string{"--with-checks", "--"}, run...), "--policy=external", "--all", "--checks=markdown-links")...)
		sees(t, out, code, 1, "missing.md", "0 of 1 checks passed")
		if strings.Contains(out, "replaces the built-in") {
			t.Errorf("an installed pinned check replaces its built-in on purpose, so no note:\n%s", out)
		}
	}
}

func Test0164_4(t *testing.T) {
	t.Parallel()
	cache := fiPrivateCache(t)
	stale := fmt.Sprintf(`{"tag":"v%s","checked":%q}`, rlCandidate(t), time.Now().Add(-time.Hour).Format(time.RFC3339))
	mustDo(t, os.WriteFile(filepath.Join(strings.TrimPrefix(cache[0], "FITNESS_CACHE_DIR="), "latest-release.json"), []byte(stale), 0o600))
	env := append(cache, rlReleases(t, http.StatusForbidden, "").env...)
	out, code := fiRun(t, pinned0164(t, `{"markdown-links": "v1.0.0"}`), env, "--with-checks", "--version", "latest", "--install-only")
	sees(t, out, code, 0, "warning: GitHub did not answer the release lookup, so no check installs; the built-in checks run instead")
	if strings.Contains(out, "fitness-check-markdown-links") {
		t.Errorf("after an unanswered lookup the installer must ask for no check:\n%s", out)
	}
}

func Test0164_5(t *testing.T) {
	t.Parallel()
	cache := fiPrivateCache(t)
	repo := pinned0164(t, `{"markdown-links": "v1.0.0"}`)
	// The engine unpacks into the cache first, so only the check install is timed.
	_, _ = fiRun(t, t.TempDir(), cache, "--install-only")
	env := append(cache, "HTTPS_PROXY="+silentListener(t), "NO_PROXY=", "no_proxy=")
	start := time.Now()
	out, code := fiRun(t, repo, env, "--with-checks", "--install-only")
	sees(t, out, code, 0, "does not download", "the built-in checks run instead")
	if took := time.Since(start); took > 5*time.Second {
		t.Errorf("behind a silent proxy the check install must give up within 5s, took %s:\n%s", took, out)
	}
	start = time.Now()
	again, _ := fiRun(t, repo, env, "--with-checks", "--install-only")
	if took := time.Since(start); took > 3*time.Second || strings.Contains(again, "for the check release index") {
		t.Errorf("a second install within the minute must not ask the silent index again, took %s:\n%s", took, again)
	}
}
