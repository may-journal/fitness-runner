package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/hooks"
)

// runInit installs the shared git hooks into .githooks and points git at them,
// so a consumer repo adopts the canonical hooks by running `fitness init`
// rather than copying files.
func runInit(_ []string) int {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := installHooks(root); err != nil {
		fmt.Fprintln(os.Stderr, "fitness init: "+err.Error())
		return 1
	}
	fmt.Println("fitness init: installed hooks to .githooks and set core.hooksPath")
	return 0
}

// installHooks sets core.hooksPath so git runs the hooks, then writes every
// embedded hook into root/.githooks. Setting the path first means a folder
// that is not a git repo is refused before anything is written.
func installHooks(root string) error {
	if out, err := exec.Command("git", "-C", root, "config", "core.hooksPath", ".githooks").CombinedOutput(); err != nil {
		return errors.New(strings.TrimSpace(string(out)))
	}
	dir := filepath.Join(root, ".githooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, name := range hooks.Names {
		if err := writeHook(dir, name); err != nil {
			return err
		}
	}
	return nil
}

// writeHook writes one embedded hook as an executable file.
func writeHook(dir, name string) error {
	data, err := hooks.Read(name)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), data, 0o755)
}
