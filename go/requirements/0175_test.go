package requirements

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// unused0175 is a time 31 days ago, past the 30 days a prune keeps an
// unused release.
var unused0175 = time.Now().Add(-31 * 24 * time.Hour)

// cache0175 is a cache holding engine bundles, check releases and check sets,
// with the engine the test runs inside one of its bundles.
type cache0175 struct {
	dir, engine, handed string
	// kept and pruned are the paths a prune must keep and delete.
	kept, pruned []string
}

// platform0175 is this machine's platform as cache folder names spell it.
var platform0175 = runtime.GOOS + "-" + runtime.GOARCH

// bundleKey0175 is the cache folder of a fake engine release whose SHA-256
// repeats digit.
func bundleKey0175(dir, digit string) string {
	return filepath.Join(dir, "fitness-0.0.1-"+platform0175+"-"+strings.Repeat(digit, 64))
}

// release0175 is the cache folder of check name's 1.0.0 release, holding
// its program.
func release0175(t *testing.T, dir, name string) string {
	t.Helper()
	release := filepath.Join(dir, "checks", "fitness-check-"+name+"-v1.0.0-"+platform0175+".tar")
	write(t, release, map[string]string{"fitness-check-" + name: script0172(name, `{"name":"`+name+`"}`)})
	return release
}

// checkSet0175 is a check set named name that links release's program.
func checkSet0175(t *testing.T, dir, name, release string) string {
	t.Helper()
	set := filepath.Join(dir, "check-sets", name)
	mustDo(t, os.MkdirAll(set, 0o700))
	entries, err := os.ReadDir(release)
	mustDo(t, err)
	mustDo(t, os.Symlink(filepath.Join(release, entries[0].Name()), filepath.Join(set, entries[0].Name())))
	return set
}

// prunable0175 builds a cache where the running engine, the handed check set
// and others went unused for 31 days. It adds fresh ones and a link out.
func prunable0175(t *testing.T) cache0175 {
	t.Helper()
	dir := t.TempDir()
	running, stale, fresh := bundleKey0175(dir, "a"), bundleKey0175(dir, "b"), bundleKey0175(dir, "c")
	for _, key := range []string{running, stale, fresh} {
		mustDo(t, os.MkdirAll(filepath.Join(key, "bundle-1"), 0o700))
	}
	placeRunner(t, filepath.Join(installOnly(t), "fitness"), filepath.Join(running, "bundle-1", "fitness"))
	handedRelease, oldRelease := release0175(t, dir, "md-files"), release0175(t, dir, "old-files")
	newRelease := release0175(t, dir, "new-files")
	handed, oldSet := checkSet0175(t, dir, "handed", handedRelease), checkSet0175(t, dir, "old", oldRelease)
	outside := t.TempDir()
	mustDo(t, os.Symlink(outside, bundleKey0175(dir, "d")))
	for _, path := range []string{running, stale, handedRelease, oldRelease, handed, oldSet, outside} {
		mustDo(t, os.Chtimes(path, unused0175, unused0175))
	}
	return cache0175{
		dir: dir, engine: filepath.Join(running, "bundle-1"), handed: handed,
		kept:   []string{running, fresh, handedRelease, newRelease, handed, outside, bundleKey0175(dir, "d")},
		pruned: []string{stale, oldRelease, oldSet},
	}
}

// pruneRun runs the cache's own engine in a repo with Markdown staged,
// handed the cache's check set.
func pruneRun(t *testing.T, c cache0175) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", nil)
	write(t, repo, map[string]string{"notes/plan.md": "# Plan\n"})
	git(t, repo, "add", "-A")
	env := []string{"FITNESS_CACHE_DIR=" + c.dir, "FITNESS_CHECKS_DIR=" + c.handed}
	return run0162(t, c.engine, repo, env, "--policy=external", "--checks=md-files")
}

// present fails unless each path is still there, links left unfollowed.
func present(t *testing.T, out string, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if _, err := os.Lstat(path); err != nil {
			t.Errorf("a prune must keep %s: %v\n%s", path, err, out)
		}
	}
}

// gone fails unless each path was deleted.
func gone(t *testing.T, out string, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if _, err := os.Lstat(path); err == nil {
			t.Errorf("a prune must delete %s:\n%s", path, out)
		}
	}
}

func Test0175_1(t *testing.T) {
	t.Parallel()
	c := prunable0175(t)
	out, code := pruneRun(t, c)
	sees(t, out, code, 1, "md-files ran", "fitness: pruned 3 cached engine and check releases unused for 30 days")
	present(t, out, c.kept...)
	gone(t, out, c.pruned...)
}

func Test0175_2(t *testing.T) {
	t.Parallel()
	c := prunable0175(t)
	out, code := pruneRun(t, c)
	sees(t, out, code, 1, "pruned 3 cached")
	release := filepath.Join(c.dir, "checks", "fitness-check-new-files-v1.0.0-"+platform0175+".tar")
	mustDo(t, os.Chtimes(release, unused0175, unused0175))
	out, code = pruneRun(t, c)
	sees(t, out, code, 1, "md-files ran")
	absentText(t, out, "pruned")
	present(t, out, release)

	// A day after the prune, the next run prunes again.
	yesterday := time.Now().Add(-25 * time.Hour).Format(time.RFC3339)
	mustDo(t, os.WriteFile(filepath.Join(c.dir, "cache-pruned.json"), []byte(`{"value":true,"checked":"`+yesterday+`"}`), 0o600))
	out, code = pruneRun(t, c)
	sees(t, out, code, 1, "fitness: pruned 1 cached")
	gone(t, out, release)
}

// absentText fails when out says text.
func absentText(t *testing.T, out, text string) {
	t.Helper()
	if strings.Contains(out, text) {
		t.Errorf("output must not say %q:\n%s", text, out)
	}
}

func Test0175_3(t *testing.T) {
	t.Parallel()
	r := serve0173(t)
	bundle := bundle0162(t)
	repo := staged0173(t, nil)
	env := []string{"FITNESS_CACHE_DIR=" + r.cache}
	out, code := run0162(t, bundle, repo, env, "prefetch")
	sees(t, out, code, 0, "installed 4 checks in ")
	fetchedOnly(t, r, out, "go-files", "md-files", "no-fields", "swift-files")
	for name := range firstIndex0173 {
		cached0173(t, r, name, "1.0.0")
	}
	out, code = run0162(t, bundle, repo, env, "--policy=external", checks0173, "--all")
	sees(t, out, code, 1, "md-files ran", "go-files ran", "swift-files ran", "no-fields ran")
	fetchedOnly(t, r, out, "go-files", "md-files", "no-fields", "swift-files")
}

func Test0175_4(t *testing.T) {
	t.Parallel()
	r := serve0173(t)
	bundle := bundle0162(t)
	repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"checkVersions": {"*": "v1"}}`})
	env := []string{"FITNESS_CACHE_DIR=" + r.cache}
	out, code := run0162(t, bundle, repo, env, "prefetch")
	sees(t, out, code, 0, "installed 4 checks in ")

	r.publish(t, firstIndex0173, time.Now().Add(-40*24*time.Hour))
	silent := append(slices.Clone(env), "HTTPS_PROXY="+silentListener(t), "NO_PROXY=", "no_proxy=")
	out, code = run0162(t, bundle, repo, silent, "--policy=external", checks0173, "--all")
	sees(t, out, code, 1, "GitHub did not answer the check release index", "so this run is offline and matches against the index cached",
		"md-files ran", "go-files ran", "swift-files ran", "no-fields ran")
	fetchedOnly(t, r, out, "go-files", "md-files", "no-fields", "swift-files")
}

// requiring0175 is a bundle holding a check named after each requirement
// list, and a repo pinning exactly those checks, so nothing downloads.
func requiring0175(t *testing.T, requires map[string]string) (bundle, repo string) {
	t.Helper()
	bundle = bundle0162(t)
	var pins []string
	for name, list := range requires {
		write(t, bundle, map[string]string{"fitness-check-" + name: script0172(name, `{"name":"`+name+`","requires":[`+list+`]}`)})
		pins = append(pins, `"`+name+`": "v1"`)
	}
	repo = example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"checkVersions": {` + strings.Join(pins, ", ") + `}}`})
	return bundle, repo
}

// online0175 serves the md-files check beside one requiring github and one
// requiring network, and records a fresh index naming all three.
func online0175(t *testing.T) *releases0173 {
	t.Helper()
	r := serve0173(t)
	checks := map[string]any{}
	for name, extra := range map[string]map[string]any{
		"md-files":  {"files": `\.md$`},
		"gh-reads":  {"requires": []string{"github"}},
		"net-reads": {"requires": []string{"network"}},
	} {
		r.archives[name] = archive0173(t, name)
		version := map[string]any{
			"tag":       "fitness-check-" + name + "-v1.0.0",
			"artifacts": map[string]any{platform0175: map[string]string{"url": r.url + "/" + name + ".tar", "sha256": fmt.Sprintf("%x", sha256.Sum256(r.archives[name]))}},
		}
		maps.Copy(version, extra)
		checks[name] = map[string]any{"majors": map[string]string{"1": "1.0.0"}, "versions": map[string]any{"1.0.0": version}}
	}
	record, err := json.Marshal(map[string]any{"value": map[string]any{"format": 1, "checks": checks}, "checked": time.Now().Format(time.RFC3339)})
	mustDo(t, err)
	mustDo(t, os.WriteFile(filepath.Join(r.cache, "check-index.json"), record, 0o600))
	return r
}

func Test0175_5(t *testing.T) {
	t.Parallel()
	r := online0175(t)
	repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"checkVersions": {"md-files": "v1", "gh-reads": "v1", "net-reads": "v1"}}`})
	write(t, repo, map[string]string{"notes/plan.md": "# Plan\n"})
	git(t, repo, "add", "-A")
	silent := []string{"FITNESS_CACHE_DIR=" + r.cache, "HTTPS_PROXY=" + silentListener(t), "NO_PROXY=", "no_proxy=", "GITHUB_API_URL=https://api.github.com"}
	out, code := run0162(t, bundle0162(t), repo, silent, "--policy=external", "--checks=md-files,gh-reads,net-reads")
	fetchedOnly(t, r, out, "md-files")
	sees(t, out, code, 1, "md-files ran",
		"Skipped (requires github, and this run is offline: api.github.com did not answer): gh-reads",
		"Skipped (requires network, and this run is offline: api.github.com did not answer): net-reads")
	absent(t, out, "gh-reads", "net-reads")
	for _, name := range []string{"gh-reads", "net-reads"} {
		if !strings.Contains(row(out, name), "skipped") {
			t.Errorf("%s must be a skipped row offline:\n%s", name, out)
		}
	}
}

func Test0175_6(t *testing.T) {
	t.Parallel()
	bundle, repo := requiring0175(t, map[string]string{"diff-reads": `"diff"`})
	env := []string{"FITNESS_CACHE_DIR=" + t.TempDir()}
	out, code := run0162(t, bundle, repo, env, "--policy=external", "--checks=diff-reads", "--all")
	sees(t, out, code, 0, "Skipped (requires diff, which a full run has none of): diff-reads")
	absent(t, out, "diff-reads")

	write(t, repo, map[string]string{"notes/plan.md": "# Plan\n"})
	git(t, repo, "add", "-A")
	out, code = run0162(t, bundle, repo, env, "--policy=external", "--checks=diff-reads")
	sees(t, out, code, 1, "diff-reads ran")
}
