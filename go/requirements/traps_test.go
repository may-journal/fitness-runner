package requirements

import (
	"encoding/json"
	"fmt"
	"maps"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// promptedRemote is the remote a prompting git asks a password for.
const promptedRemote = "https://git.example.invalid/app.git"

// sshRemote is a remote the real git reaches through ssh, so the prompting
// ssh asks the password.
const sshRemote = "git@git.example.invalid:app.git"

// promptingTools returns a PATH whose git and ssh wait for a password, failing
// at once under GIT_TERMINAL_PROMPT=0 or BatchMode=yes, and a PID file.
func promptingTools(t *testing.T, stalls ...string) (path, pidFile string) {
	t.Helper()
	real, err := exec.LookPath("git")
	mustDo(t, err)
	dir := t.TempDir()
	pidFile = filepath.Join(t.TempDir(), "prompt.pid")
	wait := "echo $$ > \"" + pidFile + "\"\nexec /bin/sleep 600\n"
	stall := ""
	if len(stalls) > 0 {
		stall = "for a in \"$@\"; do\n  case \"$a\" in " + strings.Join(stalls, "|") + ")\n" + wait + "  ;; esac\ndone\n"
	}
	write(t, dir, map[string]string{
		"git": "#!/bin/sh\n" + stall + "remote=\nfor a in \"$@\"; do\n  case \"$a\" in fetch|pull|push|clone|ls-remote) remote=${remote:-1} ;; " + sshRemote + ") remote=ssh ;; esac\ndone\n" +
			"[ \"$remote\" = ssh ] && remote=\n" +
			"[ -z \"$remote\" ] && exec \"" + real + "\" \"$@\"\n" +
			"if [ \"$GIT_TERMINAL_PROMPT\" = 0 ]; then\n" +
			"  echo \"fatal: could not read Username for '" + promptedRemote + "': terminal prompts disabled\" >&2\n  exit 128\nfi\n" +
			"printf \"Username for '" + promptedRemote + "': \" >&2\n" + wait,
		"ssh": "#!/bin/sh\nfor a in \"$@\"; do\n  case \"$a\" in *BatchMode=yes*) echo 'git@git.example.invalid: Permission denied (publickey).' >&2; exit 255 ;; esac\ndone\n" +
			"printf \"git@git.example.invalid's password: \" >&2\n" + wait,
	})
	return "PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH"), pidFile
}

// silentListener starts a TCP listener that accepts connections and never
// answers, as a proxy dropping traffic does, and returns its URL.
func silentListener(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	mustDo(t, err)
	var mu sync.Mutex
	var held []net.Conn
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			held = append(held, conn)
			mu.Unlock()
		}
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range held {
			_ = conn.Close()
		}
	})
	return "http://" + ln.Addr().String()
}

// generatedTree copies happyRepo with files written over it, adds n tracked text
// files under tree/ that no check judges, and commits them all. The files share
// one blob staged in one update-index call, so even 500,000 take seconds.
func generatedTree(t *testing.T, files map[string]string, n int) string {
	t.Helper()
	repo := example(t, "happyRepo", files)
	blob := strings.TrimSpace(gitInput(t, repo, strings.Repeat("0", 16)+"\n", "hash-object", "-w", "--stdin"))
	var index strings.Builder
	for i := range n {
		fmt.Fprintf(&index, "100644 %s\ttree/%04d/f%07d.dat\x00", blob, i/1000, i)
	}
	gitInput(t, repo, index.String(), "update-index", "--add", "-z", "--index-info")
	git(t, repo, "checkout-index", "--all", "--force", "--index")
	git(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "chore(tree): add generated files")
	return repo
}

// gitInput runs git in repo with stdin as its input and returns what it
// printed to stdout, failing the test when git fails.
func gitInput(t *testing.T, repo, stdin string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}

// treeSizesEnv names, comma-separated, the tree sizes Test0001_14 also
// times, such as 10000,100000,500000. Unset, it times none, since a large
// tree is slow to check.
const treeSizesEnv = "FITNESS_TREE_SIZES"

// timesTrees logs one Markdown table of check and commit times on a
// generatedTree of every size treeSizesEnv names.
func timesTrees(t *testing.T) {
	t.Helper()
	sizes := strings.FieldsFunc(os.Getenv(treeSizesEnv), func(r rune) bool { return r == ',' })
	if len(sizes) == 0 {
		return
	}
	cells := map[string][]string{}
	for i, size := range sizes {
		n, err := strconv.Atoi(size)
		mustDo(t, err)
		for name, took := range treeTimes(t, n) {
			if cells[name] == nil {
				cells[name] = make([]string, len(sizes))
			}
			cells[name][i] = took
		}
	}
	t.Log("\n" + timesTable(sizes, cells))
}

// treeTimes times a generatedTree of n files: each check's row on an unscoped
// run, the run itself, and the two commits of commitTimes.
func treeTimes(t *testing.T, n int) map[string]string {
	t.Helper()
	repo := generatedTree(t, staleTest(), n)
	defer os.RemoveAll(repo)
	report := filepath.Join(t.TempDir(), "report.json")
	start := time.Now()
	fitness(t, repo, nil, "--policy=external", "--all", "--json="+report)
	times := map[string]string{"(unscoped run)": millis(time.Since(start).Milliseconds(), false)}
	raw, err := os.ReadFile(report)
	mustDo(t, err)
	var r struct {
		Checks []struct {
			Name     string `json:"name"`
			Ms       int64  `json:"ms"`
			TimedOut bool   `json:"timedOut"`
		} `json:"checks"`
	}
	mustDo(t, json.Unmarshal(raw, &r))
	for _, c := range r.Checks {
		times[c.Name] = millis(c.Ms, c.TimedOut)
	}
	maps.Copy(times, commitTimes(t, repo))
	return times
}

// commitTimes makes fixingCommit, editing one test file, then productCommit,
// editing one product file, in repo and returns each commit's times as commitRows.
func commitTimes(t *testing.T, repo string) map[string]string {
	t.Helper()
	out, code, took := fixingCommit(t, repo)
	times := commitRows(t, "test commit", out, code, took)
	out, code, took = productCommit(t, repo)
	maps.Copy(times, commitRows(t, "product commit", out, code, took))
	return times
}

// timedRows are the checks whose rows each timed commit reports: those
// that compare files or read the whole repo, and requirements.
var timedRows = append([]string{"duplicate-code", "requirements"}, architectureChecks...)

// commitRows names one commit's time and each timedRows row's, all marked
// failed when they failed, and logs the output of a failed commit.
func commitRows(t *testing.T, commit, out string, code int, took int64) map[string]string {
	t.Helper()
	total := millis(took, false)
	if code != 0 {
		t.Logf("the %s did not pass:\n%s", commit, out)
		total += " failed"
	}
	times := map[string]string{"(" + commit + ")": total}
	for _, check := range timedRows {
		times[check+" ("+commit+")"] = rowTime(row(out, check))
	}
	return times
}

// rowTime is a results-table row's time as a table cell, marked when the
// check failed, or "-" when the commit showed no such row.
func rowTime(line string) string {
	took := rowMillis.FindStringSubmatch(line)
	if took == nil {
		return "-"
	}
	if strings.Contains(line, "│ failed") {
		return took[1] + " ms failed"
	}
	return took[1] + " ms"
}

// productCommit commits, after fixingCommit, the productEdit files.
func productCommit(t *testing.T, repo string) (string, int, int64) {
	t.Helper()
	return timedCommit(t, repo, installerEnv(t), productEdit(), "fix(store): say the store keeps saved records")
}

// productEdit is an edited comment in the store's product file with its test,
// doc and changelog entry.
func productEdit() map[string]string {
	store := architectedRepo()["src/store/store.go"]
	return map[string]string{
		"src/store/store.go":      strings.Replace(store, "keeps records.", "keeps saved records.", 1),
		"src/store/store_test.go": edited(testFile("store", "Test0001_1")),
		acDoc:                     strings.Replace(acDocWith("User opens the app"), "I open the app.", "I open the app and see saved records.", 1),
		"changelog/2026-10-26-1300-store-comment.md": bareEntryText("2026.10.26.1300", "Fix: the store comment says it keeps saved records."),
	}
}

// staleTest is architectedRepo with a store test named for no acceptance, so the
// requirements check fails that file until a commit fixes it.
func staleTest() map[string]string {
	files := architectedRepo()
	files["src/store/store_test.go"] = testFile("store", "Test0001_1", "Test0001_9")
	return files
}

// fixingCommit installs the hooks in a staleTest repo and commits the store test
// without its stale test, so the check judges the base.
func fixingCommit(t *testing.T, repo string) (string, int, int64) {
	t.Helper()
	env := installerEnv(t)
	got, code := fitness(t, repo, env, "init")
	sees(t, got, code, 0, "fitness init: installed hooks")
	fix := map[string]string{
		"src/store/store_test.go":                 testFile("store", "Test0001_1"),
		"changelog/2026-10-26-1200-store-test.md": bareEntryText("2026.10.26.1200", "Fix: the store test owns only its own acceptance."),
	}
	return timedCommit(t, repo, env, fix, "test(store): drop the stale store test")
}

// timedCommit commits files in repo with message and env for the hooks and
// returns the output, exit code and milliseconds.
func timedCommit(t *testing.T, repo string, env []string, files map[string]string, message string) (string, int, int64) {
	t.Helper()
	write(t, repo, files)
	user(t, repo, env, append([]string{"add"}, slices.Collect(maps.Keys(files))...)...)
	start := time.Now()
	out, code := user(t, repo, env, "commit", "-m", message)
	return out, code, time.Since(start).Milliseconds()
}

// rowMillis reads a results-table row's time, in milliseconds.
var rowMillis = regexp.MustCompile(`│ (\d+)ms\s+│\s*$`)

// millis shows ms as a table cell, marked when the check timed out.
func millis(ms int64, timedOut bool) string {
	if timedOut {
		return strconv.FormatInt(ms, 10) + " ms, timed out"
	}
	return strconv.FormatInt(ms, 10) + " ms"
}

// timesTable renders cells as a Markdown table: one row per name, sorted,
// and one column per size.
func timesTable(sizes []string, cells map[string][]string) string {
	var b strings.Builder
	b.WriteString("| check | " + strings.Join(sizes, " files | ") + " files |\n|---" + strings.Repeat("|---", len(sizes)) + "|\n")
	for _, name := range slices.Sorted(maps.Keys(cells)) {
		b.WriteString("| " + name + " | " + strings.Join(cells[name], " | ") + " |\n")
	}
	return b.String()
}
