package requirements

import (
	"regexp"
	"strings"
	"testing"
)

// coverageExclude runs `fitness-install -- vitest-coverage-exclude` on
// happyRepo with files written over it.
func coverageExclude(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "vitest-coverage-exclude")
}

// coverageExcludeRemove is the failure line naming one quoted entry.
func coverageExcludeRemove(quoted string) string {
	return "Vitest coverage exclude must be empty, so coverage judges every file; remove: " + quoted
}

// coverageExcludeConfig is a CommonJS Vitest config with exclude as given.
func coverageExcludeConfig(exclude string) string {
	return "module.exports = { test: { coverage: { exclude: " + exclude + " } } };\n"
}

// coverageExcludeOneFile matches a results row that passed judging 1 file.
var coverageExcludeOneFile = regexp.MustCompile(`│ passed\s+│ 1\s+│`)

func Test0059_1(t *testing.T) {
	out, code := coverageExclude(t, nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "vitest-coverage-exclude")
}

func Test0059_2(t *testing.T) {
	out, code := coverageExclude(t, map[string]string{
		"package.json":     "{}\n",
		"vitest.config.js": coverageExcludeConfig("[]"),
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0059_3(t *testing.T) {
	out, code := coverageExclude(t, map[string]string{
		"package.json":     "{}\n",
		"vitest.config.js": coverageExcludeConfig(`["**/*.test.ts", "dist"]`),
	})
	sees(t, out, code, 1, coverageExcludeRemove(`"**/*.test.ts"`), coverageExcludeRemove(`"dist"`))
}

func Test0059_4(t *testing.T) {
	out, code := coverageExclude(t, map[string]string{
		"package.json": `{"vitest": {"test": {"coverage": {"exclude": ["src/foo.ts"]}}}}` + "\n",
	})
	sees(t, out, code, 1, coverageExcludeRemove(`"src/foo.ts"`))
}

func Test0059_5(t *testing.T) {
	out, code := coverageExclude(t, map[string]string{
		"package.json":     `{"vitest": {"coverage": {"exclude": ["src/foo.ts"]}}}` + "\n",
		"vitest.config.js": coverageExcludeConfig("[]"),
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0059_6(t *testing.T) {
	out, code := coverageExclude(t, map[string]string{
		"package.json": "{}\n",
		"vitest.config.mjs": `import { DTS_GLOB } from "./globs.js";` + "\n" +
			`export default { test: { coverage: { exclude: [DTS_GLOB, "src/foo.ts"] } } };` + "\n",
	})
	sees(t, out, code, 1, coverageExcludeRemove(`"src/foo.ts"`))
	if strings.Contains(out, "DTS_GLOB") {
		t.Errorf("an imported name must not be reported:\n%s", out)
	}
}

func Test0059_7(t *testing.T) {
	out, code := coverageExclude(t, map[string]string{
		"package.json": "{}\n",
		"node_modules/@mayjournal/fitness-shared/config/vitest.config.js": coverageExcludeConfig(`["src/shared.ts"]`),
	})
	sees(t, out, code, 1, coverageExcludeRemove(`"src/shared.ts"`))
}

func Test0059_8(t *testing.T) {
	out, code := coverageExclude(t, map[string]string{"package.json": "{}\n"})
	sees(t, out, code, 0, "All 1 checks passed")
	if !coverageExcludeOneFile.MatchString(row(out, "vitest-coverage-exclude")) {
		t.Errorf("vitest-coverage-exclude must pass judging 1 file:\n%s", out)
	}
}
