package requirements

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// rlLookups is a saved releases API. It answers every latest release lookup
// with one status, naming one release when the status is 200, and keeps the
// Authorization header each lookup sent.
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

func Test0088_1(t *testing.T) {
	t.Parallel()
	version := rlCandidate(t)
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			lookups, gh := rlReleases(t, http.StatusOK, version), rlGh(t, "from-gh")
			out, code := rlLatest(t, fiPrivateCache(t), lookups, slices.Concat(gh.env, []string{name + "=from-" + name})...)
			sees(t, out, code, 0, "fitness: version="+version+" ")
			if got := lookups.sent(); !slices.Equal(got, []string{"Bearer from-" + name}) {
				t.Errorf("lookups sent Authorization %q, want one Bearer from-%s", got, name)
			}
			if n := gh.count("gh", "auth token"); n != 0 {
				t.Errorf("asked gh for a token %d times with %s set", n, name)
			}
		})
	}
}

func Test0088_2(t *testing.T) {
	t.Parallel()
	version := rlCandidate(t)
	lookups, gh := rlReleases(t, http.StatusOK, version), rlGh(t, "from-gh")
	out, code := rlLatest(t, fiPrivateCache(t), lookups, gh.env...)
	sees(t, out, code, 0, "fitness: version="+version+" ")
	if got := lookups.sent(); !slices.Equal(got, []string{"Bearer from-gh"}) {
		t.Errorf("lookups sent Authorization %q, want one Bearer from-gh", got)
	}
}

func Test0088_3(t *testing.T) {
	t.Parallel()
	version := rlCandidate(t)
	cache, lookups, gh := fiPrivateCache(t), rlReleases(t, http.StatusOK, version), rlGh(t, "from-gh")
	first, _ := rlLatest(t, cache, lookups, gh.env...)
	again, code := rlLatest(t, cache, lookups, gh.env...)
	sees(t, again, code, 0, "fitness: latest resolved to v"+version, "fitness: version="+version+" ")
	if fiLastLine(first) != fiLastLine(again) {
		t.Errorf("second run installed %q, first %q", fiLastLine(again), fiLastLine(first))
	}
	if n := len(lookups.sent()); n != 1 {
		t.Errorf("two runs within 10 minutes sent %d lookups, want 1", n)
	}
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

func Test0088_4(t *testing.T) {
	t.Parallel()
	version := rlCandidate(t)
	givens := map[string]func(*testing.T, []string, string){"recorded": rlRecorded, "installed": rlInstalled}
	for name, given := range givens {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cache, gh := fiPrivateCache(t), rlGh(t, "from-gh")
			given(t, cache, version)
			out, code := rlLatest(t, cache, rlReleases(t, http.StatusForbidden, version), gh.env...)
			sees(t, out, code, 0, "fitness: warning: ", "HTTP 403; using "+version, "fitness: version="+version+" ")
			if n := strings.Count(out, "warning"); n != 1 {
				t.Errorf("printed %d warnings, want 1:\n%s", n, out)
			}
		})
	}
}

// rlSilentBudget is how long a run may take when the releases API never
// answers: the five-second lookup, then an install from the private cache.
const rlSilentBudget = 10 * time.Second

func Test0088_6(t *testing.T) {
	t.Parallel()
	version := rlCandidate(t)
	cache, gh, api := fiPrivateCache(t), rlGh(t, "from-gh"), silentListener(t)
	rlRecorded(t, cache, version)
	env := slices.Concat(cache, gh.env, []string{"GITHUB_API_URL=" + api, "GH_TOKEN=", "GITHUB_TOKEN="})
	start := time.Now()
	out, code := fiRun(t, t.TempDir(), env, "--version", "latest", "--install-only")
	took := time.Since(start)
	sees(t, out, code, 0, "fitness: asking "+strings.TrimPrefix(api, "http://")+" for the latest release (gives up after 5s)",
		"fitness: warning: ", "Client.Timeout exceeded", "; using "+version, "fitness: version="+version+" ")
	if took > rlSilentBudget {
		t.Errorf("a silent releases API held the run %s, want under %s", took.Round(time.Second), rlSilentBudget)
	}
}

func Test0088_5(t *testing.T) {
	t.Parallel()
	version := rlCandidate(t)
	lookups, gh := rlReleases(t, http.StatusOK, version), rlGh(t, "from-gh")
	out, code := fiRun(t, t.TempDir(), slices.Concat(fiPrivateCache(t), lookups.env, gh.env), "--version", "v"+version, "--install-only")
	sees(t, out, code, 0, "fitness: version="+version+" ")
	if n := len(lookups.sent()); n != 0 {
		t.Errorf("an exact version sent %d lookups, want 0", n)
	}
	if n := gh.count("gh", "auth token"); n != 0 {
		t.Errorf("an exact version asked gh for a token %d times", n)
	}
}
