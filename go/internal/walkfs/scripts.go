package walkfs

import (
	"path"
	"regexp"
	"strings"
)

// ScriptExts are the TypeScript and JavaScript file extensions.
var ScriptExts = strings.Fields(".ts .tsx .mts .cts .js .jsx .mjs .cjs")

var (
	// toolConfig matches a tool's config file, such as vite.config.ts or
	// rollup.config.analyze.mjs, or a dotfile such as .eslintrc.cjs.
	toolConfig = regexp.MustCompile(`^\.|\.config\.`)
	// scriptTest matches the test files vitest and jest find by default,
	// such as adder.test.ts or adder.spec.jsx.
	scriptTest = regexp.MustCompile(`\.(?:test|spec)\.[cm]?[jt]sx?$`)
)

// Scripts lists the TypeScript and JavaScript files a project writes: every
// tracked file with a script extension, except those .gitattributes marks
// generated and tool config files such as vite.config.ts or .eslintrc.cjs.
func Scripts(root string) []string {
	var out []string
	for _, f := range WithoutGenerated(root, FilesByExt(root, ScriptExts...)) {
		if !toolConfig.MatchString(path.Base(f)) {
			out = append(out, f)
		}
	}
	return out
}

// ScriptTest reports whether file is a test file vitest or jest runs by
// default: its name ends in .test or .spec before a script extension.
func ScriptTest(file string) bool {
	return scriptTest.MatchString(path.Base(file))
}
