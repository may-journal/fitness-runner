package langs

import (
	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/nodejs"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Node is JavaScript and TypeScript: a repo with a tracked package.json.
var Node = lang.Module{
	Name:      "node",
	Detect:    nodejs.Applies,
	Exts:      walkfs.ScriptExts,
	Manifests: []string{nodejs.Manifest},
	Lockfiles: []string{
		"package-lock.json", "npm-shrinkwrap.json", "yarn.lock", "pnpm-lock.yaml",
		"bun.lock", "bun.lockb", "deno.lock",
	},
	Tools: []string{"eslint", "prettier", "biome", "vitest"},
	Checks: []string{
		"eslint", "prettier", "no-eslint-disable", "node-version",
		"dependency-currency", "vitest-coverage-exclude", "vitest-coverage-full",
	},
	Dicts: []string{"node.txt", "npm.txt", "typescript.txt"},
}
