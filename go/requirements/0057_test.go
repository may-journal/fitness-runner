package requirements

import (
	"strings"
	"testing"
)

// swiftlintRun runs `fitness-install -- swiftlint` on happyRepo with files
// written over it and env added.
func swiftlintRun(t *testing.T, files map[string]string, env ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), env, "swiftlint")
}

// swiftlintSees asserts out shows want and exit code code. The table wraps
// long absolute paths mid-word, so spaces are ignored when comparing.
func swiftlintSees(t *testing.T, out string, gotCode, code int, want string) {
	t.Helper()
	sees(t, out, gotCode, code)
	if !strings.Contains(swiftlintSquash(out), swiftlintSquash(want)) {
		t.Errorf("output is missing %q:\n%s", want, out)
	}
}

// swiftlintSquash drops table borders and all whitespace from text.
func swiftlintSquash(text string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(text, "│", " ")), "")
}

func Test0057_1(t *testing.T) {
	t.Parallel()
	swiftlintTool(t)
	out, code := swiftlintRun(t, map[string]string{"Short.swift": "let ab = 1\n"})
	swiftlintSees(t, out, code, 1,
		"Short.swift:1:5 - Variable name 'ab' should be between 3 and 40 characters long (identifier_name)")
}

func Test0057_2(t *testing.T) {
	t.Parallel()
	swiftlintTool(t)
	out, code := swiftlintRun(t, map[string]string{"NoNewline.swift": "let value = 1"})
	swiftlintSees(t, out, code, 1,
		"NoNewline.swift:1 - Files should have a single trailing newline (trailing_newline)")
}

func Test0057_3(t *testing.T) {
	t.Parallel()
	swiftlintTool(t)
	out, code := swiftlintRun(t, map[string]string{"Clean.swift": "let value = 1\n"})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0057_4(t *testing.T) {
	t.Parallel()
	swiftlintTool(t)
	out, code := swiftlintRun(t, map[string]string{
		".swiftlint.yml": "disabled_rules:\n  - identifier_name\n",
		"Short.swift":    "let ab = 1\n",
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0057_5(t *testing.T) {
	t.Parallel()
	swiftlintTool(t)
	out, code := swiftlintRun(t, nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "swiftlint")
}

func Test0057_6(t *testing.T) {
	t.Parallel()
	out, code := swiftlintRun(t, map[string]string{"Clean.swift": "let value = 1\n"}, pathWithout(t, "swiftlint"))
	sees(t, out, code, 1, "SwiftLint not installed: brew install swiftlint")
}

func Test0057_8(t *testing.T) {
	t.Parallel()
	swiftlintTool(t)
	out, code := swiftlintRun(t, map[string]string{
		".swiftlint.yml": "excluded:\n  - Short.swift\n",
		"Short.swift":    "let ab = 1\n",
	})
	sees(t, out, code, 0, "All 1 checks passed")
}
