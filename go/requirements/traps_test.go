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

// promptingTools returns a PATH setting whose git and ssh wait for a
// password as the real tools do on a terminal, and the file where the
// waiting tool writes its process ID. Git waits on any command that reaches
// a remote and runs the real git otherwise, or when named an ssh remote, so
// the real git runs ssh. Each fails at once, with the
// real tool's message, when the no-prompt setting is on: git under
// GIT_TERMINAL_PROMPT=0 and ssh under BatchMode=yes. Git also waits,
// whatever the no-prompt setting, on each subcommand stalls names, as a
// call on an index another process holds does.
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

// silentListener starts a TCP listener on this machine that accepts every
// connection and never answers, as a proxy that drops traffic does, and
// returns its address as a URL. It serves as HTTPS_PROXY or as a registry.
// The test's end closes it and every connection it holds.
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

// generatedTree copies happyRepo with files written over it, as example
// does, adds n more tracked files under tree/, and commits them all. The
// files are plain text no check judges, a thousand to a folder, and live
// only in the test's temporary folder.
func generatedTree(t *testing.T, files map[string]string, n int) string {
	t.Helper()
	repo := example(t, "happyRepo", files)
	for i := range n {
		dir := filepath.Join(repo, "tree", fmt.Sprintf("%04d", i/1000))
		if i%1000 == 0 {
			mustDo(t, os.MkdirAll(dir, 0o755))
		}
		mustDo(t, os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%07d.dat", i)), []byte(strings.Repeat("0", 16)+"\n"), 0o644))
	}
	git(t, repo, "add", "-A")
	git(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "chore(tree): add generated files")
	return repo
}

// treeSizesEnv names, comma-separated, the tree sizes Test0001_14 also
// times, such as 10000,100000,500000. Unset, it times none, since a large
// tree takes minutes to write.
const treeSizesEnv = "FITNESS_TREE_SIZES"

// timesTrees logs, as one Markdown table, how long each check takes on an
// unscoped run of a generatedTree of every size treeSizesEnv names, and how
// long a commit that edits one test file takes. Each tree is deleted once
// timed.
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

// treeTimes times one generatedTree of n files: each check's row on an
// unscoped run, the run itself, and a commit scoped to one edited test
// file, with its requirements row.
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

// commitTimes makes fixingCommit in repo and returns its time and its
// requirements row's, marked failed when the commit failed.
func commitTimes(t *testing.T, repo string) map[string]string {
	t.Helper()
	out, code, took := fixingCommit(t, repo)
	req := rowMillis.FindStringSubmatch(row(out, "requirements"))
	if code != 0 || req == nil {
		t.Log("the scoped commit did not pass:\n" + out)
		return map[string]string{"(scoped commit)": millis(took, false) + " failed", "requirements (scoped commit)": "-"}
	}
	return map[string]string{"(scoped commit)": millis(took, false), "requirements (scoped commit)": req[1] + " ms"}
}

// staleTest is architectedRepo with a store test named for no acceptance,
// so the requirements check fails the store test file until a commit fixes
// it.
func staleTest() map[string]string {
	files := architectedRepo()
	files["src/store/store_test.go"] = testFile("store", "Test0001_1", "Test0001_9")
	return files
}

// fixingCommit installs the hooks in a staleTest repo and commits, as a
// user does, the store test without its stale test and a changelog entry
// that says so. The requirements check judges the base commit to see the
// test file was failing there. It returns the output, the exit code and
// the commit's milliseconds.
func fixingCommit(t *testing.T, repo string) (string, int, int64) {
	t.Helper()
	env := installerEnv(t)
	got, code := fitness(t, repo, env, "init")
	sees(t, got, code, 0, "fitness init: installed hooks")
	fix := map[string]string{
		"src/store/store_test.go":                 testFile("store", "Test0001_1"),
		"changelog/2026-10-26-1200-store-test.md": bareEntryText("2026.10.26.1200", "Fix: the store test owns only its own acceptance."),
	}
	write(t, repo, fix)
	user(t, repo, env, append([]string{"add"}, slices.Collect(maps.Keys(fix))...)...)
	start := time.Now()
	out, code := user(t, repo, env, "commit", "-m", "test(store): drop the stale store test")
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
