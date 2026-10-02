package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/hooks"
)

func TestEmbeddedHooks(t *testing.T) {
	for _, name := range hooks.Names {
		data, err := hooks.Read(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !strings.HasPrefix(string(data), "#!/usr/bin/env bash") {
			t.Errorf("%s missing shebang", name)
		}
		if !strings.Contains(string(data), "fitness") {
			t.Errorf("%s should call fitness", name)
		}
	}
}

// assertHooksExecutable checks every hook is written under dir and executable.
func assertHooksExecutable(t *testing.T, dir string) {
	t.Helper()
	for _, name := range hooks.Names {
		info, err := os.Stat(filepath.Join(dir, ".githooks", name))
		if err != nil {
			t.Fatalf("missing hook %s: %v", name, err)
		}
		if info.Mode().Perm()&0o100 == 0 {
			t.Errorf("%s is not executable (%v)", name, info.Mode())
		}
	}
}

func TestInstallHooks(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")

	if err := installHooks(dir); err != nil {
		t.Fatal(err)
	}

	assertHooksExecutable(t, dir)

	// git now points at .githooks.
	out, err := exec.Command("git", "-C", dir, "config", "core.hooksPath").Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out)); got != ".githooks" {
		t.Errorf("core.hooksPath = %q, want .githooks", got)
	}
}
