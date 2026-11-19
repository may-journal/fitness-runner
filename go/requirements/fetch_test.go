// cspell:ignore Typeflag

package requirements

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// moduleFree starts the import path of every package in this repo's module.
const moduleFree = "github.com/may-journal/fitness-runner/go/"

// checkPackageFree reports whether path is a check's package.
func checkPackageFree(path string) bool {
	return strings.Contains(path, "/cmd/fitness-check-")
}

// goListFree runs go list with args in go/ and returns its output.
func goListFree(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("go", append([]string{"list"}, args...)...)
	cmd.Dir = ".."
	out, err := cmd.Output()
	mustDo(t, err)
	return string(out)
}

// fileImportsFree returns the import paths of one Go file.
func fileImportsFree(t *testing.T, file string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	mustDo(t, err)
	var paths []string
	for _, spec := range parsed.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		mustDo(t, err)
		paths = append(paths, path)
	}
	return paths
}

// engineImportsFree returns the imports of the engine's files apart from
// checks.go, failing each check among them. It also counts the checks that
// checks.go imports.
func engineImportsFree(t *testing.T) (imports []string, builtIns int) {
	t.Helper()
	for _, file := range strings.Fields(goListFree(t, "-f", `{{range .GoFiles}}{{$.Dir}}/{{.}}{{"\n"}}{{end}}{{range .IgnoredGoFiles}}{{$.Dir}}/{{.}}{{"\n"}}{{end}}`, "./cmd/fitness")) {
		paths := fileImportsFree(t, file)
		if filepath.Base(file) == "checks.go" {
			builtIns = len(slices.DeleteFunc(paths, func(p string) bool { return !checkPackageFree(p) }))
			continue
		}
		for _, path := range slices.DeleteFunc(slices.Clone(paths), func(p string) bool { return !checkPackageFree(p) }) {
			t.Errorf("engine file %s imports check package %s", filepath.Base(file), path)
		}
		// Only this module's packages can reach a check, and go list resolves
		// them on any platform, unlike a package for another OS.
		imports = append(imports, slices.DeleteFunc(paths, func(p string) bool { return !strings.HasPrefix(p, moduleFree) })...)
	}
	return slices.Compact(slices.Sorted(slices.Values(imports))), builtIns
}

// innerDiscovery is a Go test that runs the fitness named by REQ_FITNESS, as a
// repo's own requirement tests do. It fails unless that run used a check.
const innerDiscovery = `package m

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

const fresh = FRESH

func TestInnerFitness(t *testing.T) {
	out, _ := exec.Command(os.Getenv("REQ_FITNESS"), "--policy=external", "--checks=markdown-links").CombinedOutput()
	if strings.Contains(string(out), "no result from the check") || !strings.Contains(string(out), "All 1 checks passed") {
		t.Fatalf("fitness inside the test found no installed check:\n%s", out)
	}
}
`

// goTestOnThisEngine builds fitness-check-go-test from fitness-checks on this
// checkout's engine code. It returns a checks folder that holds that build
// in place of the released one.
func goTestOnThisEngine(t *testing.T) string {
	t.Helper()
	here, err := os.Getwd()
	mustDo(t, err)
	module := filepath.Join(filepath.Dir(here))
	build := t.TempDir()
	write(t, build, map[string]string{
		"go.mod": "module example.com/build\n\ngo 1.26\n\n" +
			"replace github.com/may-journal/fitness-runner/go => " + module + "\n",
		"tools.go": "//go:build tools\n\npackage build\n\nimport _ \"github.com/may-journal/fitness-checks/cmd/fitness-check-go-test\"\n",
	})
	checks := t.TempDir()
	for _, args := range [][]string{
		{"get", "github.com/may-journal/fitness-checks@main"},
		{"build", "-o", filepath.Join(checks, "fitness-check-go-test"), "github.com/may-journal/fitness-checks/cmd/fitness-check-go-test"},
	} {
		cmd := exec.Command("go", args...)
		cmd.Dir, cmd.Env = build, append(os.Environ(), "GOFLAGS=-mod=mod")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	released, err := os.ReadDir(os.Getenv("FITNESS_CHECKS_DIR"))
	mustDo(t, err)
	for _, e := range released {
		if e.Name() != "fitness-check-go-test" {
			mustDo(t, os.Symlink(filepath.Join(os.Getenv("FITNESS_CHECKS_DIR"), e.Name()), filepath.Join(checks, e.Name())))
		}
	}
	return checks
}

// scriptDiscovery is a check program whose --describe prints describe and whose
// run prints a failing result saying said.
func scriptDiscovery(describe, said string) string {
	return "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '" + describe + "'; exit 0; }\n" +
		"echo '{\"ok\":false,\"errors\":[\"" + said + "\"],\"filesChecked\":1}'\nexit 1\n"
}

// bundleDiscovery places the installed runner alone in a fresh folder, as a clean
// install of the engine leaves it, and returns the folder.
func bundleDiscovery(t *testing.T) string {
	t.Helper()
	bundle := t.TempDir()
	placeRunner(t, filepath.Join(installOnly(t), "fitness"), filepath.Join(bundle, "fitness"))
	return bundle
}

// runDiscovery runs the fitness in bundle in repo with env added.
func runDiscovery(t *testing.T, bundle, repo string, env []string, args ...string) (string, int) {
	t.Helper()
	return execute(t, func() *exec.Cmd {
		cmd := exec.Command(filepath.Join(bundle, "fitness"), args...)
		// Only the folders a test names hold checks, not the suite's installed ones.
		cmd.Dir, cmd.Env = repo, append(append(userEnv(), "FITNESS_CHECKS_DIR="), env...)
		return cmd
	})
}

// pinLinesPins keeps the `<name> <pin>` lines of out, dropping the
// installer's own lines.
func pinLinesPins(out string) string {
	var kept []string
	for _, line := range strings.Split(out, "\n") {
		if fields := strings.Fields(line); len(fields) == 2 && strings.HasPrefix(fields[1], "v") && !strings.Contains(line, ":") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

// installedInInstall matches the line naming the folder of installed checks.
var installedInInstall = regexp.MustCompile(`installed (\d+) checks in (\S+)`)

// tagsInstall serves tags as the check release tag lookup. It returns the
// environment pointing the installer at it, with a private cache and no token.
func tagsInstall(t *testing.T, tags ...string) []string {
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

// pinnedInstall is a repo whose checkVersions holds pins.
func pinnedInstall(t *testing.T, pins string) string {
	t.Helper()
	return example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"checkVersions": ` + pins + `}`})
}

// timesClean matches a duration as the results table and summary print it.
var timesClean = regexp.MustCompile(`[0-9]+(\.[0-9]+)?(ms|s)\b`)

// resultsClean is the results table and summary of a run, line by line, with
// times and the padding they set blanked.
func resultsClean(out string) string {
	start := strings.Index(out, "┌")
	end := strings.Index(out, "Slowest:")
	if start < 0 || end < start {
		return "no results table in:\n" + out
	}
	lines := strings.Split(timesClean.ReplaceAllString(out[start:end], ""), "\n")
	for i, line := range lines {
		// A shorter time pads its cell, so runs of spaces count as one.
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	return strings.Join(lines, "\n")
}

// cleanHomeFetch is the user's environment with an empty home, and no Fitness
// cache or installed checks handed down.
func cleanHomeFetch(t *testing.T) []string {
	t.Helper()
	env := slices.DeleteFunc(userEnv(), func(kv string) bool {
		return strings.HasPrefix(kv, "FITNESS_CACHE_DIR=") || strings.HasPrefix(kv, "FITNESS_CHECKS_DIR=") || strings.HasPrefix(kv, "XDG_CACHE_HOME=")
	})
	return append(env, "HOME="+t.TempDir())
}

// fieldsOnDemand are the index fields of each fake check. One is from before
// checks declared fields, and yaml-files is new in a later index.
var fieldsOnDemand = map[string]map[string]string{
	"md-files":    {"files": `\.md$`},
	"go-files":    {"files": `\.go$`, "exclude": `(^|/)vendor/`},
	"swift-files": {"files": `\.swift$`},
	"no-fields":   {},
	"yaml-files":  {"files": `\.ya?ml$`},
}

// firstIndexOnDemand is the version of each check the first index publishes.
var firstIndexOnDemand = map[string]string{"md-files": "1.0.0", "go-files": "1.0.0", "swift-files": "1.0.0", "no-fields": "1.0.0"}

// pinsOnDemand pins every fake check at its major.
const pinsOnDemand = `{"checkVersions": {"md-files": "v1", "go-files": "v1", "swift-files": "v1", "no-fields": "v1"}}`

// checksOnDemand is the --checks flag that runs every fake check.
const checksOnDemand = "--checks=md-files,go-files,swift-files,no-fields"

// releasesOnDemand is the fake checks' server, which records which ones a run
// downloads, and the private cache whose index names them.
type releasesOnDemand struct {
	url, cache string
	archives   map[string][]byte
	mu         sync.Mutex
	fetched    []string
}

// downloaded lists the checks downloaded so far, in name order.
func (r *releasesOnDemand) downloaded() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Sorted(slices.Values(r.fetched))
}

// archiveOnDemand is a check release archive holding the program of name.
func archiveOnDemand(t *testing.T, name string) []byte {
	t.Helper()
	program := []byte(scriptApplies(name, fmt.Sprintf(`{"name":%q}`, name)))
	var buf bytes.Buffer
	w := tar.NewWriter(&buf)
	mustDo(t, w.WriteHeader(&tar.Header{Name: "fitness-check-" + name, Mode: 0o755, Size: int64(len(program)), Typeflag: tar.TypeReg}))
	_, err := w.Write(program)
	mustDo(t, err)
	mustDo(t, w.Close())
	return buf.Bytes()
}

// serveOnDemand serves the fake checks and seeds a fresh release index naming
// them, with their fields, into a private cache. A name in tampered serves
// bytes its SHA-256 does not match.
func serveOnDemand(t *testing.T, tampered ...string) *releasesOnDemand {
	t.Helper()
	r := &releasesOnDemand{cache: t.TempDir(), archives: map[string][]byte{}}
	for name := range fieldsOnDemand {
		r.archives[name] = archiveOnDemand(t, name)
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
	r.publish(t, firstIndexOnDemand, time.Now())
	return r
}

// publish records in r's cache a release index read at checked. It names
// each check in versions at its version, the newest of major 1.
func (r *releasesOnDemand) publish(t *testing.T, versions map[string]string, checked time.Time) {
	t.Helper()
	checks := map[string]any{}
	for name, number := range versions {
		version := map[string]any{
			"tag":       "fitness-check-" + name + "-v" + number,
			"artifacts": map[string]any{runtime.GOOS + "-" + runtime.GOARCH: map[string]string{"url": r.url + "/" + name + ".tar", "sha256": fmt.Sprintf("%x", sha256.Sum256(r.archives[name]))}},
		}
		for key, value := range fieldsOnDemand[name] {
			version[key] = value
		}
		checks[name] = map[string]any{"majors": map[string]string{"1": number}, "versions": map[string]any{number: version}}
	}
	record, err := json.Marshal(map[string]any{"value": map[string]any{"format": 1, "checks": checks}, "checked": checked.Format(time.RFC3339)})
	mustDo(t, err)
	mustDo(t, os.WriteFile(filepath.Join(r.cache, "check-index.json"), record, 0o600))
}

// stagedOnDemand is a repo pinning every fake check, with files staged.
func stagedOnDemand(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": pinsOnDemand})
	write(t, repo, files)
	git(t, repo, "add", "-A")
	return repo
}

// fetchRunOnDemand runs the engine in repo with r's cache, as a clean machine
// with no checks handed down.
func fetchRunOnDemand(t *testing.T, r *releasesOnDemand, repo string, args ...string) (string, int) {
	t.Helper()
	return runDiscovery(t, bundleDiscovery(t), repo, []string{"FITNESS_CACHE_DIR=" + r.cache}, append([]string{"--policy=external", checksOnDemand}, args...)...)
}

// fetchedOnly fails unless r downloaded exactly want.
func fetchedOnly(t *testing.T, r *releasesOnDemand, out string, want ...string) {
	t.Helper()
	if got := r.downloaded(); !slices.Equal(got, slices.Sorted(slices.Values(want))) {
		t.Errorf("the run must download only %q, got %q:\n%s", want, got, out)
	}
}

// hookedOnDemand is the environment the shared hooks give fitness-install, with
// a private cache that holds the installed engine and r's index.
func hookedOnDemand(t *testing.T, r *releasesOnDemand) []string {
	t.Helper()
	cache := fiPrivateCache(t)
	mustDo(t, os.Rename(filepath.Join(r.cache, "check-index.json"), filepath.Join(strings.TrimPrefix(cache[0], "FITNESS_CACHE_DIR="), "check-index.json")))
	return append(cache, "FITNESS_CHECKS_DIR=", "FITNESS_WITH_CHECKS=pinned")
}

// cachedOnDemand fails unless r's cache holds check name's program at version.
func cachedOnDemand(t *testing.T, r *releasesOnDemand, name, version string) {
	t.Helper()
	tag := "fitness-check-" + name + "-v" + version
	if _, err := os.Stat(filepath.Join(r.cache, "checks", tag+"-"+runtime.GOOS+"-"+runtime.GOARCH+".tar", "fitness-check-"+name)); err != nil {
		t.Errorf("the cache must hold %s %s: %v", name, version, err)
	}
}

// unusedCache is a time 31 days ago, past the 30 days a prune keeps an
// unused release.
var unusedCache = time.Now().Add(-31 * 24 * time.Hour)

// cacheCache is a cache holding engine bundles, check releases and check sets,
// with the engine the test runs inside one of its bundles.
type cacheCache struct {
	dir, engine, handed string
	// settings and skills point into two bundles from outside the cache.
	settings, skills string
	// kept and pruned are the paths a prune must keep and delete.
	kept, pruned []string
}

// platformCache is this machine's platform as cache folder names spell it.
var platformCache = runtime.GOOS + "-" + runtime.GOARCH

// bundleKeyCache is the cache folder of a fake engine release whose SHA-256
// repeats digit.
func bundleKeyCache(dir, digit string) string {
	return filepath.Join(dir, "fitness-0.0.1-"+platformCache+"-"+strings.Repeat(digit, 64))
}

// releaseCache is the cache folder of check name's 1.0.0 release, holding
// its program.
func releaseCache(t *testing.T, dir, name string) string {
	t.Helper()
	release := filepath.Join(dir, "checks", "fitness-check-"+name+"-v1.0.0-"+platformCache+".tar")
	write(t, release, map[string]string{"fitness-check-" + name: scriptApplies(name, `{"name":"`+name+`"}`)})
	return release
}

// checkSetCache is a check set named name that links release's program.
func checkSetCache(t *testing.T, dir, name, release string) string {
	t.Helper()
	set := filepath.Join(dir, "check-sets", name)
	mustDo(t, os.MkdirAll(set, 0o700))
	entries, err := os.ReadDir(release)
	mustDo(t, err)
	mustDo(t, os.Symlink(filepath.Join(release, entries[0].Name()), filepath.Join(set, entries[0].Name())))
	return set
}

// prunableCache builds a cache where the running engine, the handed check set
// and others went unused for 31 days. It adds fresh ones and a link out.
func prunableCache(t *testing.T) cacheCache {
	t.Helper()
	dir := t.TempDir()
	running, stale, fresh, linked := bundleKeyCache(dir, "a"), bundleKeyCache(dir, "b"), bundleKeyCache(dir, "c"), bundleKeyCache(dir, "e")
	skilled, dropped := bundleKeyCache(dir, "f"), bundleKeyCache(dir, "9")
	for _, key := range []string{running, stale, fresh, linked, skilled, dropped} {
		mustDo(t, os.MkdirAll(filepath.Join(key, "bundle-1", "skills", "fitness"), 0o700))
	}
	placeRunner(t, filepath.Join(installOnly(t), "fitness"), filepath.Join(running, "bundle-1", "fitness"))
	settings, skills := pointersCache(t, dir, linked, skilled)
	handedRelease, oldRelease := releaseCache(t, dir, "md-files"), releaseCache(t, dir, "old-files")
	newRelease := releaseCache(t, dir, "new-files")
	handed, oldSet := checkSetCache(t, dir, "handed", handedRelease), checkSetCache(t, dir, "old", oldRelease)
	halfMade := filepath.Join(dir, "check-set-123")
	mustDo(t, os.MkdirAll(halfMade, 0o700))
	outside := t.TempDir()
	mustDo(t, os.Symlink(outside, bundleKeyCache(dir, "d")))
	for _, path := range []string{running, stale, linked, skilled, dropped, handedRelease, oldRelease, handed, oldSet, halfMade, outside} {
		mustDo(t, os.Chtimes(path, unusedCache, unusedCache))
	}
	indexAtCache(t, dir, time.Now())
	return cacheCache{
		dir: dir, engine: filepath.Join(running, "bundle-1"), handed: handed, settings: settings, skills: skills,
		kept:   []string{running, fresh, linked, skilled, handedRelease, newRelease, handed, outside, bundleKeyCache(dir, "d")},
		pruned: []string{stale, dropped, oldRelease, oldSet, halfMade},
	}
}

// pointersCache makes and returns a settings file naming bundle settingsAt and
// a skills folder linking into skillsAt, as fitness-install leaves them.
func pointersCache(t *testing.T, cache, settingsAt, skillsAt string) (settings, skills string) {
	t.Helper()
	settings, skills = filepath.Join(t.TempDir(), "settings.json"), t.TempDir()
	hook := `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"'` + filepath.Join(settingsAt, "bundle-1", "fitness") + `' claude-hook"}]}]}}`
	mustDo(t, os.WriteFile(settings, []byte(hook), 0o600))
	mustDo(t, os.Symlink(filepath.Join(skillsAt, "bundle-1", "skills", "fitness"), filepath.Join(skills, "fitness")))
	recorded, err := json.Marshal([]string{settings, skills})
	mustDo(t, err)
	mustDo(t, os.WriteFile(filepath.Join(cache, "cache-pointers.json"), recorded, 0o600))
	return settings, skills
}

// pruneDue makes the next run prune, as a day after the last prune.
func pruneDue(t *testing.T, cache string) {
	t.Helper()
	yesterday := time.Now().Add(-25 * time.Hour).Format(time.RFC3339)
	mustDo(t, os.WriteFile(filepath.Join(cache, "cache-pruned.json"), []byte(`{"value":true,"checked":"`+yesterday+`"}`), 0o600))
}

// indexAtCache records in cache an empty release index read at checked. A
// prune runs only while GitHub answers, so it needs a fresh one.
func indexAtCache(t *testing.T, cache string, checked time.Time) {
	t.Helper()
	record := `{"value":{"format":1,"checks":{}},"checked":"` + checked.Format(time.RFC3339) + `"}`
	mustDo(t, os.WriteFile(filepath.Join(cache, "check-index.json"), []byte(record), 0o600))
}

// pruneRun runs the cache's own engine in a repo with Markdown staged,
// handed the cache's check set.
func pruneRun(t *testing.T, c cacheCache, extra ...string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", nil)
	write(t, repo, map[string]string{"notes/plan.md": "# Plan\n"})
	git(t, repo, "add", "-A")
	env := append([]string{"FITNESS_CACHE_DIR=" + c.dir, "FITNESS_CHECKS_DIR=" + c.handed}, extra...)
	return runDiscovery(t, c.engine, repo, env, "--policy=external", "--checks=md-files")
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

// absentText fails when out says text.
func absentText(t *testing.T, out, text string) {
	t.Helper()
	if strings.Contains(out, text) {
		t.Errorf("output must not say %q:\n%s", text, out)
	}
}

// requiringCache is a bundle holding a check named after each requirement
// list, and a repo pinning exactly those checks, so nothing downloads.
func requiringCache(t *testing.T, requires map[string]string) (bundle, repo string) {
	t.Helper()
	bundle = bundleDiscovery(t)
	var pins []string
	for name, list := range requires {
		write(t, bundle, map[string]string{"fitness-check-" + name: scriptApplies(name, `{"name":"`+name+`","requires":[`+list+`]}`)})
		pins = append(pins, `"`+name+`": "v1"`)
	}
	repo = example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"checkVersions": {` + strings.Join(pins, ", ") + `}}`})
	return bundle, repo
}

// onlineCache serves md-files and checks requiring github, network or
// neither, and records a fresh index naming all four.
func onlineCache(t *testing.T) *releasesOnDemand {
	t.Helper()
	r := serveOnDemand(t)
	checks := map[string]any{}
	for name, extra := range map[string]map[string]any{
		"md-files":  {"files": `\.md$`},
		"gh-reads":  {"requires": []string{"github"}},
		"net-reads": {"requires": []string{"network"}},
		"plain":     {},
	} {
		r.archives[name] = archiveOnDemand(t, name)
		version := map[string]any{
			"tag":       "fitness-check-" + name + "-v1.0.0",
			"artifacts": map[string]any{platformCache: map[string]string{"url": r.url + "/" + name + ".tar", "sha256": fmt.Sprintf("%x", sha256.Sum256(r.archives[name]))}},
		}
		maps.Copy(version, extra)
		checks[name] = map[string]any{"majors": map[string]string{"1": "1.0.0"}, "versions": map[string]any{"1.0.0": version}}
	}
	record, err := json.Marshal(map[string]any{"value": map[string]any{"format": 1, "checks": checks}, "checked": time.Now().Format(time.RFC3339)})
	mustDo(t, err)
	mustDo(t, os.WriteFile(filepath.Join(r.cache, "check-index.json"), record, 0o600))
	return r
}

// twoDaysCache is a time two days ago. A prune keeps an unused release 30
// days, and a superseded one a day.
var twoDaysCache = time.Now().Add(-48 * time.Hour)

// releaseAtCache is the cache folder of check name's release at version,
// holding its program, last used at when.
func releaseAtCache(t *testing.T, dir, name, version string, when time.Time) string {
	t.Helper()
	release := filepath.Join(dir, "checks", "fitness-check-"+name+"-v"+version+"-"+platformCache+".tar")
	write(t, release, map[string]string{"fitness-check-" + name: scriptApplies(name, `{"name":"`+name+`"}`)})
	mustDo(t, os.Chtimes(release, when, when))
	return release
}

// supersededCacheCache is a cache where older versions of the running
// engine, another engine and several checks sit beside newer cached ones.
type supersededCacheCache struct {
	cacheCache
	running, olderEngine, oldCheck, majorSet, pinned, recent, live, pinnedSet string
}

// supersededCache builds a cache where engine 0.0.0 and 0.0.1, the running
// one, sit beside 0.0.2, and old-files 1.0.0, pinned-files 1.0.0 and
// recent-files 1.0.0 beside 1.1.0. A check set pins pinned-files exactly,
// recent-files ran an hour ago, and a set used an hour ago links live-files.
func supersededCache(t *testing.T) supersededCacheCache {
	t.Helper()
	dir := t.TempDir()
	key := func(version, digit string) string {
		key := filepath.Join(dir, "fitness-"+version+"-"+platformCache+"-"+strings.Repeat(digit, 64))
		mustDo(t, os.MkdirAll(filepath.Join(key, "bundle-1"), 0o700))
		return key
	}
	running, older, newer := key("0.0.1", "a"), key("0.0.0", "b"), key("0.0.2", "c")
	placeRunner(t, filepath.Join(installOnly(t), "fitness"), filepath.Join(running, "bundle-1", "fitness"))
	handedRelease := releaseAtCache(t, dir, "md-files", "1.0.0", time.Now())
	handed := checkSetCache(t, dir, "handed", handedRelease)
	oldCheck := releaseAtCache(t, dir, "old-files", "1.0.0", twoDaysCache)
	oldMinor := releaseAtCache(t, dir, "old-files", "1.1.0", twoDaysCache)
	oldMajor := releaseAtCache(t, dir, "old-files", "2.0.0", time.Now())
	// A set made for a `v1` pin links old-files 1.0.0, which pins no release.
	majorSet := checkSetCache(t, dir, "major", oldCheck)
	mustDo(t, os.WriteFile(majorSet+".pins", []byte("old-files v1\n"), 0o600))
	pinned := releaseAtCache(t, dir, "pinned-files", "1.0.0", twoDaysCache)
	pinnedSet := checkSetCache(t, dir, "pinned", pinned)
	mustDo(t, os.WriteFile(pinnedSet+".pins", []byte("pinned-files v1.0.0\n"), 0o600))
	recent := releaseAtCache(t, dir, "recent-files", "1.0.0", time.Now().Add(-time.Hour))
	live := releaseAtCache(t, dir, "live-files", "1.0.0", twoDaysCache)
	liveSet := checkSetCache(t, dir, "live", live)
	mustDo(t, os.Chtimes(liveSet, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour)))
	newest := []string{releaseAtCache(t, dir, "pinned-files", "1.1.0", time.Now()), releaseAtCache(t, dir, "recent-files", "1.1.0", time.Now()), releaseAtCache(t, dir, "live-files", "1.1.0", time.Now())}
	for _, path := range []string{running, older, majorSet, pinnedSet} {
		mustDo(t, os.Chtimes(path, twoDaysCache, twoDaysCache))
	}
	indexAtCache(t, dir, time.Now())
	return supersededCacheCache{
		cacheCache: cacheCache{
			dir: dir, engine: filepath.Join(running, "bundle-1"), handed: handed,
			kept:   append([]string{running, newer, handedRelease, handed, oldMinor, oldMajor, pinned, pinnedSet, recent, live, liveSet}, newest...),
			pruned: []string{older, oldCheck, majorSet},
		},
		running: running, olderEngine: older, oldCheck: oldCheck, majorSet: majorSet, pinned: pinned, recent: recent, live: live, pinnedSet: pinnedSet,
	}
}

// handedCache is the release folder the cache's handed check set links.
func handedCache(c cacheCache) string {
	return filepath.Join(c.dir, "checks", "fitness-check-md-files-v1.0.0-"+platformCache+".tar")
}
