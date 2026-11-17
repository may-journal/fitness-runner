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
	running, stale, fresh, linked := bundleKey0175(dir, "a"), bundleKey0175(dir, "b"), bundleKey0175(dir, "c"), bundleKey0175(dir, "e")
	for _, key := range []string{running, stale, fresh, linked} {
		mustDo(t, os.MkdirAll(filepath.Join(key, "bundle-1"), 0o700))
	}
	placeRunner(t, filepath.Join(installOnly(t), "fitness"), filepath.Join(running, "bundle-1", "fitness"))
	// Claude Code settings or linked skills point into this bundle.
	mustDo(t, os.WriteFile(filepath.Join(linked, "linked"), nil, 0o600))
	handedRelease, oldRelease := release0175(t, dir, "md-files"), release0175(t, dir, "old-files")
	newRelease := release0175(t, dir, "new-files")
	handed, oldSet := checkSet0175(t, dir, "handed", handedRelease), checkSet0175(t, dir, "old", oldRelease)
	halfMade := filepath.Join(dir, "check-set-123")
	mustDo(t, os.MkdirAll(halfMade, 0o700))
	outside := t.TempDir()
	mustDo(t, os.Symlink(outside, bundleKey0175(dir, "d")))
	for _, path := range []string{running, stale, linked, handedRelease, oldRelease, handed, oldSet, halfMade, outside} {
		mustDo(t, os.Chtimes(path, unused0175, unused0175))
	}
	indexAt0175(t, dir, time.Now())
	return cache0175{
		dir: dir, engine: filepath.Join(running, "bundle-1"), handed: handed,
		kept:   []string{running, fresh, linked, handedRelease, newRelease, handed, outside, bundleKey0175(dir, "d")},
		pruned: []string{stale, oldRelease, oldSet, halfMade},
	}
}

// indexAt0175 records in cache an empty release index read at checked. A
// prune runs only while GitHub answers, so it needs a fresh one.
func indexAt0175(t *testing.T, cache string, checked time.Time) {
	t.Helper()
	record := `{"value":{"format":1,"checks":{}},"checked":"` + checked.Format(time.RFC3339) + `"}`
	mustDo(t, os.WriteFile(filepath.Join(cache, "check-index.json"), []byte(record), 0o600))
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
	sees(t, out, code, 1, "md-files ran", "fitness: pruned 4 cached engine and check releases unused for 30 days")
	present(t, out, c.kept...)
	gone(t, out, c.pruned...)
}

func Test0175_2(t *testing.T) {
	t.Parallel()
	c := prunable0175(t)
	out, code := pruneRun(t, c)
	sees(t, out, code, 1, "pruned 4 cached")
	release := filepath.Join(c.dir, "checks", "fitness-check-new-files-v1.0.0-"+platform0175+".tar")
	mustDo(t, os.Chtimes(release, unused0175, unused0175))
	out, code = pruneRun(t, c)
	sees(t, out, code, 1, "md-files ran")
	absentText(t, out, "pruned")
	present(t, out, release)

	// A day after the prune, a run prunes again, but only while GitHub
	// answers the index read. Offline, a deleted check could not return.
	yesterday := time.Now().Add(-25 * time.Hour).Format(time.RFC3339)
	mustDo(t, os.WriteFile(filepath.Join(c.dir, "cache-pruned.json"), []byte(`{"value":true,"checked":"`+yesterday+`"}`), 0o600))
	indexAt0175(t, c.dir, time.Now().Add(-11*time.Minute))
	out, code = pruneRun(t, c)
	sees(t, out, code, 1, "md-files ran")
	absentText(t, out, "pruned")
	present(t, out, release)
	indexAt0175(t, c.dir, time.Now())
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
	_, set, _ := strings.Cut(out, "installed 4 checks in ")
	set, _, _ = strings.Cut(set, "\n")
	pins, err := os.ReadFile(set + ".pins")
	mustDo(t, err)
	for name := range firstIndex0173 {
		if !strings.Contains(string(pins), name+" v") {
			t.Errorf("the check set must record the pin of %s:\n%s", name, pins)
		}
	}
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
	sees(t, out, code, 1, "GitHub did not answer the check release index", "so this run works offline against the index cached",
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
	sees(t, out, code, 0, "Skipped (requires diff, which only a run of changed files has): diff-reads")
	absent(t, out, "diff-reads")

	write(t, repo, map[string]string{"notes/plan.md": "# Plan\n"})
	git(t, repo, "add", "-A")
	out, code = run0162(t, bundle, repo, env, "--policy=external", "--checks=diff-reads")
	sees(t, out, code, 1, "diff-reads ran")
}

// twoDays0175 is a time two days ago. A prune keeps an unused release 30
// days, and a superseded one a day.
var twoDays0175 = time.Now().Add(-48 * time.Hour)

// releaseAt0175 is the cache folder of check name's release at version,
// holding its program, last used at when.
func releaseAt0175(t *testing.T, dir, name, version string, when time.Time) string {
	t.Helper()
	release := filepath.Join(dir, "checks", "fitness-check-"+name+"-v"+version+"-"+platform0175+".tar")
	write(t, release, map[string]string{"fitness-check-" + name: script0172(name, `{"name":"`+name+`"}`)})
	mustDo(t, os.Chtimes(release, when, when))
	return release
}

// supersededCache0175 is a cache where older versions of the running
// engine, another engine and several checks sit beside newer cached ones.
type supersededCache0175 struct {
	cache0175
	running, olderEngine, oldCheck, pinned, recent, pinnedSet string
}

// superseded0175 builds a cache where engine 0.0.0 and 0.0.1, the running
// one, sit beside 0.0.2, and old-files 1.0.0, pinned-files 1.0.0 and
// recent-files 1.0.0 beside 1.1.0. A check set pins pinned-files exactly, and
// recent-files ran an hour ago.
func superseded0175(t *testing.T) supersededCache0175 {
	t.Helper()
	dir := t.TempDir()
	key := func(version, digit string) string {
		key := filepath.Join(dir, "fitness-"+version+"-"+platform0175+"-"+strings.Repeat(digit, 64))
		mustDo(t, os.MkdirAll(filepath.Join(key, "bundle-1"), 0o700))
		return key
	}
	running, older, newer := key("0.0.1", "a"), key("0.0.0", "b"), key("0.0.2", "c")
	placeRunner(t, filepath.Join(installOnly(t), "fitness"), filepath.Join(running, "bundle-1", "fitness"))
	handedRelease := releaseAt0175(t, dir, "md-files", "1.0.0", time.Now())
	handed := checkSet0175(t, dir, "handed", handedRelease)
	oldCheck := releaseAt0175(t, dir, "old-files", "1.0.0", twoDays0175)
	oldMinor := releaseAt0175(t, dir, "old-files", "1.1.0", twoDays0175)
	oldMajor := releaseAt0175(t, dir, "old-files", "2.0.0", time.Now())
	// A set made for a `v1` pin links old-files 1.0.0, which pins no release.
	majorSet := checkSet0175(t, dir, "major", oldCheck)
	mustDo(t, os.WriteFile(majorSet+".pins", []byte("old-files v1\n"), 0o600))
	pinned := releaseAt0175(t, dir, "pinned-files", "1.0.0", twoDays0175)
	pinnedSet := checkSet0175(t, dir, "pinned", pinned)
	mustDo(t, os.WriteFile(pinnedSet+".pins", []byte("pinned-files v1.0.0\n"), 0o600))
	recent := releaseAt0175(t, dir, "recent-files", "1.0.0", time.Now().Add(-time.Hour))
	newest := []string{releaseAt0175(t, dir, "pinned-files", "1.1.0", time.Now()), releaseAt0175(t, dir, "recent-files", "1.1.0", time.Now())}
	for _, path := range []string{running, older, majorSet, pinnedSet} {
		mustDo(t, os.Chtimes(path, twoDays0175, twoDays0175))
	}
	indexAt0175(t, dir, time.Now())
	return supersededCache0175{
		cache0175: cache0175{
			dir: dir, engine: filepath.Join(running, "bundle-1"), handed: handed,
			kept:   append([]string{running, newer, handedRelease, handed, oldMinor, oldMajor, majorSet, pinned, pinnedSet, recent}, newest...),
			pruned: []string{older, oldCheck},
		},
		running: running, olderEngine: older, oldCheck: oldCheck, pinned: pinned, recent: recent, pinnedSet: pinnedSet,
	}
}

func Test0175_7(t *testing.T) {
	t.Parallel()
	t.Run("superseded versions go", func(t *testing.T) {
		t.Parallel()
		c := superseded0175(t)
		out, code := pruneRun(t, c.cache0175)
		sees(t, out, code, 1, "md-files ran", "fitness: pruned 2 cached engine and check releases unused for 30 days or superseded")
		gone(t, out, c.olderEngine, c.oldCheck)
		present(t, out, c.kept...)
	})
	t.Run("exactly pinned and recently used versions stay", func(t *testing.T) {
		t.Parallel()
		c := superseded0175(t)
		out, code := pruneRun(t, c.cache0175)
		sees(t, out, code, 1, "pruned 2 cached")
		present(t, out, c.pinned, c.recent)

		// Once no set pins it exactly and a day passes unused, each goes too.
		mustDo(t, os.WriteFile(c.pinnedSet+".pins", []byte("pinned-files v1\n"), 0o600))
		mustDo(t, os.Chtimes(c.recent, twoDays0175, twoDays0175))
		yesterday := time.Now().Add(-25 * time.Hour).Format(time.RFC3339)
		mustDo(t, os.WriteFile(filepath.Join(c.dir, "cache-pruned.json"), []byte(`{"value":true,"checked":"`+yesterday+`"}`), 0o600))
		out, code = pruneRun(t, c.cache0175)
		sees(t, out, code, 1, "fitness: pruned 2 cached")
		gone(t, out, c.pinned, c.recent)
	})
	t.Run("the running engine stays", func(t *testing.T) {
		t.Parallel()
		c := superseded0175(t)
		out, code := pruneRun(t, c.cache0175)
		sees(t, out, code, 1, "md-files ran", "pruned 2 cached")
		present(t, out, c.running, c.engine)
		out, code = pruneRun(t, c.cache0175)
		sees(t, out, code, 1, "md-files ran")
	})
}
