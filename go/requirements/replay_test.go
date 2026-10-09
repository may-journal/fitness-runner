package requirements

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// standIns are the external programs fitness calls that tests replace with
// saved responses: GitHub's CLI and the JavaScript and Swift tools.
var standIns = []string{"gh", "eslint", "prettier", "vitest", "swiftlint"}

// TestMain lets this test binary stand in for those programs. Run under one
// of their names, it answers from saved responses instead of running tests.
func TestMain(m *testing.M) {
	if name := filepath.Base(os.Args[0]); slices.Contains(standIns, name) {
		os.Exit(replay(name, os.Args[1:]))
	}
	os.Exit(m.Run())
}

// response is one saved answer: when every Match string appears in the
// call's arguments, the stand-in prints Stdout and Stderr and exits with
// Exit. An empty Match answers any call.
type response struct {
	Match  []string `json:"match"`
	Stdout string   `json:"stdout"`
	Stderr string   `json:"stderr"`
	Exit   int      `json:"exit"`
}

// replay answers one call from $FITNESS_REPLAY/<name>.json, the first
// response that matches, and logs the call to $FITNESS_REPLAY/<name>.log.
func replay(name string, args []string) int {
	dir := os.Getenv("FITNESS_REPLAY")
	call := strings.Join(args, " ")
	logCall(filepath.Join(dir, name+".log"), call)
	var saved []response
	raw, _ := os.ReadFile(filepath.Join(dir, name+".json"))
	_ = json.Unmarshal(raw, &saved)
	for _, r := range saved {
		if matches(call, r.Match) {
			fmt.Print(r.Stdout)
			fmt.Fprint(os.Stderr, r.Stderr)
			return r.Exit
		}
	}
	fmt.Fprintf(os.Stderr, "%s stand-in: no saved response for: %s\n", name, call)
	return 97
}

// matches reports whether every want appears in call.
func matches(call string, want []string) bool {
	for _, w := range want {
		if !strings.Contains(call, w) {
			return false
		}
	}
	return true
}

// logCall appends one call, newline-separated, to path.
func logCall(path, call string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintln(f, strings.ReplaceAll(call, "\n", `\n`))
}

// replays is a test's set of stand-ins: the environment that puts them first
// on PATH, and the folder holding their saved responses and call logs.
type replays struct {
	env []string
	dir string
}

// standIn puts a stand-in for each named program first on PATH, answering
// from its saved responses, and returns the environment to run fitness with.
func standIn(t *testing.T, saved map[string][]response) replays {
	t.Helper()
	self, err := os.Executable()
	mustDo(t, err)
	dir, bin := t.TempDir(), t.TempDir()
	for name, responses := range saved {
		data, err := json.Marshal(responses)
		mustDo(t, err)
		mustDo(t, os.WriteFile(filepath.Join(dir, name+".json"), data, 0o644))
		mustDo(t, os.Symlink(self, filepath.Join(bin, name)))
	}
	path := bin + string(os.PathListSeparator) + os.Getenv("PATH")
	return replays{env: []string{"PATH=" + path, "FITNESS_REPLAY=" + dir}, dir: dir}
}

// calls returns every call the named stand-in received, in order.
func (r replays) calls(name string) []string {
	raw, _ := os.ReadFile(filepath.Join(r.dir, name+".log"))
	return strings.FieldsFunc(string(raw), func(c rune) bool { return c == '\n' })
}

// called reports whether some call to name holds every want.
func (r replays) called(name string, want ...string) bool {
	for _, c := range r.calls(name) {
		if matches(c, want) {
			return true
		}
	}
	return false
}

// registry serves saved npm registry answers, package name to response body,
// and returns its URL; a missing package answers 404.
func registry(t *testing.T, packages map[string]string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := packages[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}
