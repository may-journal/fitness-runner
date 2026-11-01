package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/may-journal/fitness-runner/go/internal/hooks"
)

// runInit points git at the repo's .fitness-hooks, else installs the shared
// hooks into .githooks, so no repo copies hooks.
func runInit(_ []string) int {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if shippedHooksDir(root) == syncedHooksDir {
		return useSyncedHooks(root)
	}
	if err := installHooks(root); err != nil {
		fmt.Fprintln(os.Stderr, "fitness init: "+err.Error())
		return 1
	}
	fmt.Println("fitness init: installed hooks to .githooks and set core.hooksPath")
	return 0
}

// useSyncedHooks points core.hooksPath at the .fitness-hooks folder the repo
// ships and writes nothing, since the org sync owns those files.
func useSyncedHooks(root string) int {
	if err := setHooksPath(root, syncedHooksDir); err != nil {
		fmt.Fprintln(os.Stderr, "fitness init: "+err.Error())
		return 1
	}
	fmt.Println("fitness init: set core.hooksPath to " + syncedHooksDir + ", which the repo ships")
	return 0
}

// installHooks sets core.hooksPath so git runs the hooks, then writes every
// embedded hook into root/.githooks. Setting the path first means a folder
// that is not a git repo is refused before anything is written.
func installHooks(root string) error {
	if err := setHooksPath(root, installedHooksDir); err != nil {
		return err
	}
	dir := filepath.Join(root, installedHooksDir)
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
