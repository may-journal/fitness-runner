package node

import (
	"path"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// ScriptExts are the TypeScript and JavaScript file extensions.
var ScriptExts = strings.Fields(".ts .tsx .mts .cts .js .jsx .mjs .cjs")

var (
	// toolConfig matches a tool's config file, such as vite.config.ts or
	// rollup.config.analyze.mjs, or a dotfile such as .eslintrc.cjs.
	toolConfig = regexp.MustCompile(`^\.|\.config\.`)
	// scriptTest matches the test files vitest and jest find by default,
	// such as adder.test.ts, adder.spec.jsx, or any script in a __tests__
	// folder.
	scriptTest = regexp.MustCompile(`\.(?:test|spec)\.[cm]?[jt]sx?$|(?:^|/)__tests__/`)
)

// Scripts lists the TypeScript and JavaScript files a project writes: every
// tracked file with a script extension, except those .gitattributes marks
// generated and tool config files such as vite.config.ts or .eslintrc.cjs.
func Scripts(root string) []string {
	var out []string
	for _, f := range walkfs.WithoutGenerated(root, walkfs.FilesByExt(root, ScriptExts...)) {
		if !ToolConfig(f) {
			out = append(out, f)
		}
	}
	return out
}

// ToolConfig reports whether file is a tool's config file by its name,
// such as vite.config.ts, rollup.config.mjs, or .eslintrc.cjs.
func ToolConfig(file string) bool {
	return toolConfig.MatchString(path.Base(file))
}

// ScriptTest reports whether file is a test file vitest or jest runs by
// default: its name ends in .test or .spec before a script extension, or it
// sits in a __tests__ folder.
func ScriptTest(file string) bool {
	return scriptTest.MatchString(file)
}
