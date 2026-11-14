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
	if strings.Contains(again, "downloading fitness-check-") {
		t.Errorf("a second install must reuse the cached checks:\n%s", again)
	}
}

func Test0164_2(t *testing.T) {
	t.Parallel()
	env := tags0164(t, "fitness-check-gofmt-v1.0.0")
	out, code := fiRun(t, pinned0164(t, `{"gofmt": "v2"}`), env, "--with-checks", "--install-only")
	sees(t, out, code, 1, "no release of check gofmt matches v2")
}

func Test0164_3(t *testing.T) {
	t.Parallel()
	repo := pinned0164(t, `{"markdown-links": "v1.0.0"}`)
	commit(t, repo, map[string]string{"broken.md": "# Broken\n\nSee [the guide](missing.md).\n"}, "docs(broken): link a missing guide")
	out, code := fiRun(t, repo, tags0164(t), "--with-checks", "--", "--no-builtins", "--policy=external", "--all", "--checks=markdown-links")
	sees(t, out, code, 1, "missing.md", "0 of 1 checks passed")
}
