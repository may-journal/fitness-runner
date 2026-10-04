package main

import (
	"os"
	"strings"
)

// runnerEnvironment exposes verified tools before caller tools, preserving
// project tools and other settings without changing the installer environment.
func runnerEnvironment(bin string) []string {
	const key = "PATH="
	entry := key + bin
	if path := os.Getenv("PATH"); path != "" {
		entry += string(os.PathListSeparator) + path
	}
	env := os.Environ()
	for i, value := range env {
		if strings.HasPrefix(value, key) {
			env[i] = entry
			return env
		}
	}
	return append(env, entry)
}
