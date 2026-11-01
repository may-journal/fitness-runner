// cspell:ignore mvnw gradlew

package requirements

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// standIns are the external programs tests replace with saved responses: gh,
// the JavaScript and Swift tools, the JDK, Java build tools and Docker.
var standIns = []string{"gh", "eslint", "prettier", "biome", "vitest", "swiftlint", "java", "mvn", "mvnw", "gradle", "gradlew", "docker"}

// TestMain lets this test binary stand in for those programs. Run under one
// of their names, it answers from saved responses instead of running tests.
func TestMain(m *testing.M) {
	if name := filepath.Base(os.Args[0]); slices.Contains(standIns, name) {
		os.Exit(replay(name, os.Args[1:]))
	}
	if name := filepath.Base(os.Args[0]); strings.HasPrefix(name, timedPrefix) {
		os.Exit(timedCheck(name, os.Args[1:]))
	}
	keepHostGitOut()
	os.Exit(m.Run())
}

// repoGitConfig is the git config every test repo gets, whatever the host's,
// since a signed tag needs a message no test gives. Background maintenance after
// a commit can still be writing a pack when the test removes the repo.
var repoGitConfig = [][2]string{{"tag.gpgSign", "false"}, {"maintenance.auto", "false"}, {"gc.auto", "0"}}

// keepHostGitOut puts repoGitConfig in the environment, so every git the
// tests or fitness start reads it.
func keepHostGitOut() {
	for i, kv := range repoGitConfig {
		_ = os.Setenv(fmt.Sprintf("GIT_CONFIG_KEY_%d", i), kv[0])
		_ = os.Setenv(fmt.Sprintf("GIT_CONFIG_VALUE_%d", i), kv[1])
	}
	_ = os.Setenv("GIT_CONFIG_COUNT", fmt.Sprint(len(repoGitConfig)))
}

// response is one saved answer: when every Match string appears in the call's
// arguments, the stand-in writes Files, prints Stdout and exits.
type response struct {
	Match  []string          `json:"match"`
	Files  map[string]string `json:"files"`
	Stdout string            `json:"stdout"`
	Stderr string            `json:"stderr"`
	Exit   int               `json:"exit"`
	// Wait, when set, is how many seconds the stand-in hangs first, as a
	// stalled service does.
	Wait int `json:"wait"`
	// Token, when set, also requires the call's GH_TOKEN to equal it.
	Token string `json:"token"`
}

// replay answers one call from $FITNESS_REPLAY/<name>.json, the first
// response that matches, and logs the call to $FITNESS_REPLAY/<name>.log.
// A call that reads its input from stdin logs that input after its arguments.
func replay(name string, args []string) int {
	dir := os.Getenv("FITNESS_REPLAY")
	call := strings.Join(args, " ")
	if strings.Contains(call, "--input -") {
		sent, _ := io.ReadAll(os.Stdin)
		call += " " + string(sent)
	}
	logCall(filepath.Join(dir, name+".log"), call)
	var saved []response
	raw, _ := os.ReadFile(filepath.Join(dir, name+".json"))
	_ = json.Unmarshal(raw, &saved)
	for _, r := range saved {
		if r.answers(call) {
			time.Sleep(time.Duration(r.Wait) * time.Second)
			writeFiles(r.Files)
			fmt.Print(r.Stdout)
			fmt.Fprint(os.Stderr, r.Stderr)
			return r.Exit
		}
	}
	if pr, ok := eventPR(name, call); ok {
		fmt.Println(pr)
		return 0
	}
	fmt.Fprintf(os.Stderr, "%s stand-in: no saved response for: %s\n", name, call)
	return 97
}

// eventPR answers a gh read of the event's pull request with the event's
// number, title and body unless a saved response overrides it.
func eventPR(name, call string) (string, bool) {
	pr := savedEventPR()
	if name != "gh" || pr == nil {
		return "", false
	}
	path := fmt.Sprintf("repos/%s/pulls/%d", os.Getenv("GITHUB_REPOSITORY"), pr.Number)
	if !strings.HasPrefix(call, "api ") || !strings.HasSuffix(call, " "+path) {
		return "", false
	}
	out, err := json.Marshal(pr)
	return string(out), err == nil
}

// eventPRFields are the pull request fields a gh read of one PR returns.
type eventPRFields struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// savedEventPR is the pull request the saved Actions event carries, or nil
// when there is no event or it carries none.
func savedEventPR() *eventPRFields {
	var ev struct {
		PR *eventPRFields `json:"pull_request"`
	}
	raw, err := os.ReadFile(os.Getenv("GITHUB_EVENT_PATH"))
	if err != nil || json.Unmarshal(raw, &ev) != nil {
		return nil
	}
	return ev.PR
}

// writeFiles writes each file a saved answer leaves behind, such as a test
// report, relative to the folder the stand-in runs in.
func writeFiles(files map[string]string) {
	for name, body := range files {
		_ = os.MkdirAll(filepath.Dir(name), 0o755)
		_ = os.WriteFile(name, []byte(body), 0o644)
	}
}

// answers reports whether r answers call: every Match string appears in it,
// and it runs with r's Token, when set, as GH_TOKEN.
func (r response) answers(call string) bool {
	return matches(call, r.Match) && (r.Token == "" || os.Getenv("GH_TOKEN") == r.Token)
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

// replays is a test's stand-ins: the environment putting them first on PATH and
// the folder holding their saved responses and call logs.
type replays struct {
	env []string
	dir string
}

// standIn puts a stand-in for each named program first on PATH, answering from
// saved responses, and returns the environment to run fitness with.
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
	return r.count(name, want...) > 0
}

// count returns how many calls to name hold every want.
func (r replays) count(name string, want ...string) int {
	n := 0
	for _, c := range r.calls(name) {
		if matches(c, want) {
			n++
		}
	}
	return n
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
