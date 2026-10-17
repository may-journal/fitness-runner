package langs

import (
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/node"
)

// Node is JavaScript and TypeScript: a repo with a tracked package.json.
var Node = lang.Module{
	Name:      "node",
	Detect:    node.Applies,
	Exts:      node.ScriptExts,
	Manifests: []string{node.Manifest},
	Lockfiles: []string{
		"package-lock.json", "npm-shrinkwrap.json", "yarn.lock", "pnpm-lock.yaml",
		"bun.lock", "bun.lockb", "deno.lock",
	},
	Tools: []string{"eslint", "prettier", "biome", "vitest"},
	Checks: []string{
		"eslint", "prettier", "no-eslint-disable", "node-version",
		"dependency-currency", "vitest-coverage-exclude", "vitest-coverage-full",
	},
	Dicts:           []string{"node.txt", "npm.txt", "typescript.txt"},
	Technology:      node.Technology,
	TechnologyWords: strings.Fields("node node.js nodejs javascript typescript bun deno"),
	Runtime:         node.Runtime,
	Tests:           node.Tests,
	Scans: map[string][]string{
		"build-output-untracked":   node.TypeScriptExts,
		"repeated-string-literals": node.SourceExts,
	},
	ToolConfig: node.ToolConfig,
	TestFile:   node.ScriptTest,
}
