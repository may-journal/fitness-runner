package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// runnerEnvironment exposes verified tools before caller tools, preserving
// project tools and other settings without changing the installer environment.
// Installed checks reach the engine through FITNESS_CHECKS_DIR.
func runnerEnvironment(bin, checks string) []string {
	const key = "PATH="
	entry := key + bin
	if path := os.Getenv("PATH"); path != "" {
		entry += string(os.PathListSeparator) + path
	}
	env := os.Environ()
	env = withChecksDir(env, checks)
	if filepath.Base(os.Args[0]) == "pre-commit" {
		env = append(env, "FITNESS_INTERNAL_HOOK=pre-commit")
	}
	for i, value := range env {
		if strings.HasPrefix(value, key) {
			env[i] = entry
			return env
		}
	}
	return append(env, entry)
}

// withChecksDir sets FITNESS_CHECKS_DIR in env to checks, leaving env as it
// is when no checks were installed.
func withChecksDir(env []string, checks string) []string {
	if checks == "" {
		return env
	}
	return append(slices.DeleteFunc(env, func(kv string) bool { return strings.HasPrefix(kv, "FITNESS_CHECKS_DIR=") }), "FITNESS_CHECKS_DIR="+checks)
}
