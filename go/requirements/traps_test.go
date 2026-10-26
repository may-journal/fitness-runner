package requirements

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// promptedRemote is the remote a prompting git asks a password for.
const promptedRemote = "https://git.example.invalid/app.git"

// promptingTools returns a PATH setting whose git and ssh wait for a
// password as the real tools do on a terminal, and the file where the
// waiting tool writes its process ID. Git waits on any command that reaches
// a remote and runs the real git otherwise. Each fails at once, with the
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
		"git": "#!/bin/sh\n" + stall + "remote=\nfor a in \"$@\"; do\n  case \"$a\" in fetch|pull|push|clone|ls-remote) remote=1 ;; esac\ndone\n" +
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
