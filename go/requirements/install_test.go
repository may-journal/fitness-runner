package requirements

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// fiRun runs the candidate `fitness-install <args>` in dir with env added
// and returns its output and exit code.
func fiRun(t *testing.T, dir string, env []string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(installer(t), args...)
	cmd.Dir, cmd.Env = dir, append(userEnv(), env...)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		t.Fatalf("running fitness-install: %v", err)
	}
	return ansi.ReplaceAllString(string(out), ""), cmd.ProcessState.ExitCode()
}

// fiLastLine is the last printed line, where install-only names its folder.
func fiLastLine(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// fiFile returns the text of a file the installer wrote.
func fiFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	mustDo(t, err)
	return string(data)
}

// fiVersion matches the release line the installer prints before installing.
var fiVersion = regexp.MustCompile(`fitness: version=(\S+) `)

// fiPrivateCache makes a cache holding only the verified release archive,
// copied from the cache the environment already filled, so nothing downloads.
func fiPrivateCache(t *testing.T) []string {
	t.Helper()
	out, code := fiRun(t, t.TempDir(), nil, "--install-only")
	if code != 0 {
		t.Fatalf("installing fitness:\n%s", out)
	}
	key := filepath.Dir(fiLastLine(out))
	cache := t.TempDir()
	copied := filepath.Join(cache, filepath.Base(key))
	mustDo(t, os.MkdirAll(copied, 0o700))
	mustDo(t, os.WriteFile(filepath.Join(copied, "archive.tar.gz"), []byte(fiFile(t, filepath.Join(key, "archive.tar.gz"))), 0o600))
	return []string{"FITNESS_CACHE_DIR=" + cache}
}

// fiAction is a GitHub job's environment, writing its files under dir.
func fiAction(dir string, inputs ...string) []string {
	return append([]string{
		"GITHUB_ACTIONS=true",
		"GITHUB_STEP_SUMMARY=" + filepath.Join(dir, "summary"),
		"GITHUB_OUTPUT=" + filepath.Join(dir, "output"),
		"GITHUB_PATH=" + filepath.Join(dir, "path"),
	}, inputs...)
}

// rlLookups is a saved releases API answering latest release lookups with
// one status, naming a release on 200, and keeping each Authorization header.
type rlLookups struct {
	env  []string
	mu   sync.Mutex
	auth []string
}

// rlReleases serves lookups answering status with release version, and
// returns them with the environment pointing the installer at them. The
// environment sets no token variable.
func rlReleases(t *testing.T, status int, version string) *rlLookups {
	t.Helper()
	lookups := &rlLookups{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lookups.mu.Lock()
		lookups.auth = append(lookups.auth, r.Header.Get("Authorization"))
		lookups.mu.Unlock()
		w.WriteHeader(status)
		if status != http.StatusOK {
			fmt.Fprint(w, `{"message":"API rate limit exceeded"}`)
			return
		}
		fmt.Fprintf(w, `{"tag_name":"v%s"}`, version)
	}))
	t.Cleanup(srv.Close)
	lookups.env = []string{"GITHUB_API_URL=" + srv.URL, "GH_TOKEN=", "GITHUB_TOKEN="}
	return lookups
}

// sent returns the Authorization header of every lookup, in order.
func (l *rlLookups) sent() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.auth)
}

// rlDefault matches the version --help names as the installer's default.
var rlDefault = regexp.MustCompile(`exact release version, or explicit latest \(default "([^"]+)"\)`)

// rlCandidate is the release version the candidate installer carries.
func rlCandidate(t *testing.T) string {
	t.Helper()
	out, _ := fiRun(t, t.TempDir(), nil, "--help")
	match := rlDefault.FindStringSubmatch(out)
	if match == nil {
		t.Fatalf("no default version in --help:\n%s", out)
	}
	return match[1]
}

// rlGh is a logged-in gh that prints token for `gh auth token`.
func rlGh(t *testing.T, token string) replays {
	t.Helper()
	return standIn(t, map[string][]response{"gh": {{Match: []string{"auth token"}, Stdout: token + "\n"}}})
}

// rlLatest runs `fitness-install --version latest --install-only` with the
// private cache, the saved lookups, and env added.
func rlLatest(t *testing.T, cache []string, lookups *rlLookups, env ...string) (string, int) {
	t.Helper()
	return fiRun(t, t.TempDir(), slices.Concat(cache, lookups.env, env), "--version", "latest", "--install-only")
}

// rlRecorded writes a lookup of version recorded an hour ago into the
// private cache, past the 10 minutes it stands for.
func rlRecorded(t *testing.T, cache []string, version string) {
	t.Helper()
	dir := strings.TrimPrefix(cache[0], "FITNESS_CACHE_DIR=")
	record := fmt.Sprintf(`{"tag":"v%s","checked":%q}`, version, time.Now().Add(-time.Hour).Format(time.RFC3339))
	mustDo(t, os.WriteFile(filepath.Join(dir, "latest-release.json"), []byte(record), 0o600))
}

// rlInstalled installs version into the private cache by exact version.
func rlInstalled(t *testing.T, cache []string, version string) {
	t.Helper()
	if out, code := fiRun(t, t.TempDir(), cache, "--version", version, "--install-only"); code != 0 {
		t.Fatalf("installing %s:\n%s", version, out)
	}
}

// rlSilentBudget is how long a run may take when the releases API never
// answers: the five-second lookup, then a private-cache install.
const rlSilentBudget = 10 * time.Second

// stripRun writes files over happyRepo, runs `fitness strip-front-matter`
// there, and returns the repo, what it printed, and its exit code.
func stripRun(t *testing.T, files map[string]string) (string, string, int) {
	t.Helper()
	repo := example(t, "happyRepo", files)
	out, code := fitness(t, repo, nil, "strip-front-matter")
	return repo, out, code
}

// holds asserts the file at rel in repo reads exactly want.
func holds(t *testing.T, repo, rel, want string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
	mustDo(t, err)
	if string(got) != want {
		t.Errorf("%s reads %q, want %q", rel, got, want)
	}
}

// stripBody is a doc body that opens with a rule and a list, so a strip that
// eats more than the block shows.
const stripBody = "# Guide\n\n---\n\n- relatedConfigurations: stays in the body.\n"

func mustLink(t *testing.T, path string) string {
	t.Helper()
	target, err := os.Readlink(path)
	mustDo(t, err)
	return target
}

// skillLink matches each Markdown link target in a skill.
var skillLink = regexp.MustCompile(`\]\(([^)\s]+)\)`)

// resolves fails the test when what a skill points the reader at is missing.
func resolves(t *testing.T, path, what string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("%s does not resolve: %v", what, err)
	}
}

// checkNames lists every check by the name a user passes to fitness.
func checkNames(t *testing.T) []string {
	t.Helper()
	folders, err := filepath.Glob(filepath.Join("..", "cmd", "fitness-check-*"))
	mustDo(t, err)
	if len(folders) == 0 {
		t.Fatal("no check folders found under go/cmd")
	}
	names := make([]string, 0, len(folders))
	for _, folder := range folders {
		names = append(names, strings.TrimPrefix(filepath.Base(folder), "fitness-check-"))
	}
	return names
}

// onboardingSkill links the skills into a new folder with the installer, as
// an agent does, and returns the onboarding skill's text.
func onboardingSkill(t *testing.T) string {
	t.Helper()
	folder := filepath.Join(t.TempDir(), "skills")
	out, code := fiRun(t, t.TempDir(), nil, "--link-skills", folder)
	sees(t, out, code, 0, "fitness: skill "+filepath.Join(folder, "onboarding")+" linked to ")
	return fiFile(t, filepath.Join(folder, "onboarding", "SKILL.md"))
}

// onboardingLink matches a link into the repo's main branch, with its optional heading.
var onboardingLink = regexp.MustCompile(`\]\(https://github\.com/may-journal/fitness-runner/blob/main/([^)#]+)(?:#([^)]+))?\)`)

// slugDrop holds what GitHub removes from a heading to name its anchor.
var slugDrop = regexp.MustCompile(`[^a-z0-9_ -]`)

// headingSlug is how GitHub names a heading's anchor.
func headingSlug(heading string) string {
	kept := slugDrop.ReplaceAllString(strings.ToLower(strings.TrimSpace(heading)), "")
	return strings.ReplaceAll(kept, " ", "-")
}

// hasHeading reports whether a markdown file has a heading with that anchor.
// A line inside a fenced code block is code, such as a shell comment, not a heading.
func hasHeading(text, anchor string) bool {
	fenced := false
	for _, line := range strings.Split(text, "\n") {
		fenced = fenced != isFence(line)
		if !fenced && isHeading(line, anchor) {
			return true
		}
	}
	return false
}

// isFence reports whether the line opens or closes a fenced code block.
func isFence(line string) bool {
	trimmed := strings.TrimLeft(line, " ")
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

func isHeading(line, anchor string) bool {
	return strings.HasPrefix(line, "#") && headingSlug(strings.TrimLeft(line, "# ")) == anchor
}

// checkSkillLink fails unless the linked file exists in the repo and holds the
// linked heading.
func checkSkillLink(t *testing.T, path, anchor string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("%s does not exist in the repo: %v", path, err)
	}
	if anchor != "" && !hasHeading(string(data), anchor) {
		t.Fatalf("%s has no heading for #%s", path, anchor)
	}
}

// Issue URLs the gh stand-in hands out for the migrated ADRs.
const (
	sqliteIssue = "https://github.com/may-journal/example/issues/41"
	syncIssue   = "https://github.com/may-journal/example/issues/42"
	filesIssue  = "https://github.com/may-journal/example/issues/43"
	oldIssue    = "https://github.com/may-journal/example/issues/44"
	cacheIssue  = "https://github.com/may-journal/example/issues/45"
)

// oldRule is a superseded check README titled with the ADR word, its folder
// holding no program.
const oldRule = "# ADR 0002: Old rule\n\nAn old rule.\n\nStatus: Superseded by [small files](../adr-0001/README.md)\n\n" +
	"## Context\n\nIt was slow.\n\n## Decision\n\nKeep it.\n\n## Consequences\n\nIt stayed slow.\n"

// migrateRepo has three ADRs in an `adr` folder, the second superseded by
// the first and the third rejected, beside the folder's template. An index
// and a guide link them, and two check READMEs hold their decisions.
func migrateRepo() map[string]string {
	return with(heldCheck(), map[string]string{
		"docs/architecture/adr/0001-use-sqlite.md": "# ADR 0001 — Use SQLite\n\nDate: 2024-05-01\n\nStatus: accepted\n\n## Context\n\nWe sync offline.\n\n" +
			"## Decision\n\nStore notes in SQLite, synced as [ADR 0002](0002-sync.md) says.\n\n## Consequences\n\nOne file holds every note.\n",
		"docs/architecture/adr/0002-sync.md": "# ADR 0002 — Sync whole files\n\n> Whole files sync simply.\n\n" +
			"* Status: superseded by [ADR 0001](/docs/architecture/adr/0001-use-sqlite.md)\n\n" +
			"## context\n\nNotes change offline.\n\n## decision\n\nSync whole files.\n\n## consequences\n\nTwo edits clash.\n",
		"docs/architecture/adr/0003-cache.md": "# ADR 0003 — Cache notes\n\nStatus: rejected\n\n## Context\n\nReads are slow.\n\n" +
			"## Decision\n\nCache every note.\n\n## Consequences\n\nThe cache goes stale.\n",
		"docs/architecture/adr/template.md": "# ADR NNNN — Title\n\nStatus: proposed\n\n## Context\n\n## Decision\n\n## Consequences\n",
		"docs/architecture/adr/README.md":   "# ADRs\n\n- [Use SQLite](0001-use-sqlite.md)\n- [Sync](0002-sync.md)\n",
		"docs/guide.md":                     "# Guide\n\nRead [the storage decision](architecture/adr/0001-use-sqlite.md#decision) and [small files](../fitness/adr-0001/).\n",
		"fitness/adr-0002/README.md":        oldRule,
	})
}

// migrateGitHub is the saved GitHub for a migration: the ADR issues listed,
// and new ones opened in turn.
func migrateGitHub(listed string) []response {
	return []response{
		{Match: []string{"repo view"}, Stdout: "may-journal/example\n"},
		{Match: []string{"issue list", "--label ADR"}, Stdout: listed},
		{Match: []string{"label create ADR"}},
		{Match: []string{"issue create", "[ADR] Use SQLite"}, Stdout: sqliteIssue + "\n"},
		{Match: []string{"issue create", "[ADR] Sync whole files"}, Stdout: syncIssue + "\n"},
		{Match: []string{"issue create", "[ADR] Small files"}, Stdout: filesIssue + "\n"},
		{Match: []string{"issue create", "[ADR] Old rule"}, Stdout: oldIssue + "\n"},
		{Match: []string{"issue create", "[ADR] Cache notes"}, Stdout: cacheIssue + "\n"},
		{Match: []string{"issue edit"}},
		{Match: []string{"issue close"}},
	}
}

// listedIssues is the `gh issue list` output for ADR issues, each URL with
// the file its marker names, the body before it, and its state.
func listedIssues(t *testing.T, issues ...[4]string) string {
	t.Helper()
	list := []map[string]string{}
	for _, i := range issues {
		list = append(list, map[string]string{"url": i[0], "body": i[2] + "\n\n<!-- fitness:adr-migrate:" + i[1] + " -->", "state": i[3]})
	}
	out, err := json.Marshal(list)
	mustDo(t, err)
	return string(out) + "\n"
}

// keptSQLite is the listed issue an earlier migration opened for the first
// ADR, still linking the second ADR's file on the branch.
func keptSQLite(t *testing.T) [4]string {
	t.Helper()
	return [4]string{sqliteIssue, "docs/architecture/adr/0001-use-sqlite.md",
		"Synced as [ADR 0002](https://github.com/may-journal/example/blob/main/docs/architecture/adr/0002-sync.md#decision).", "OPEN"}
}

// migrate runs `fitness adr-migrate` with args in repo against saved.
func migrate(t *testing.T, repo string, saved []response, args ...string) (string, int, replays) {
	t.Helper()
	rp := standIn(t, map[string][]response{"gh": saved})
	out, code := fitness(t, repo, rp.env, append([]string{"adr-migrate"}, args...)...)
	return out, code, rp
}

// wantCall asserts some gh call holds every want, naming what it shows.
func wantCall(t *testing.T, rp replays, what string, want ...string) {
	t.Helper()
	if !rp.called("gh", want...) {
		t.Errorf("want %s, got gh calls %q", what, rp.calls("gh"))
	}
}

// wantFiles asserts each file in repo holds its text.
func wantFiles(t *testing.T, repo string, want map[string]string) {
	t.Helper()
	for name, text := range want {
		got, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(name)))
		if err != nil || string(got) != text {
			t.Errorf("%s = %q (%v), want %q", name, got, err, text)
		}
	}
}
