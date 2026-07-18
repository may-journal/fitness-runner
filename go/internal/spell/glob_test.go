package spell

// cspell:ignore modulesx doublestar axyb

import "testing"

func TestIgnoreMatcher(t *testing.T) {
	cases := []struct {
		name     string
		patterns []string
		path     string
		want     bool
	}{
		{"bare name matches at root", []string{"node_modules"}, "node_modules", true},
		{"bare name matches any depth", []string{"node_modules"}, "a/node_modules/x.md", true},
		{"bare name is not a prefix match", []string{"node_modules"}, "node_modulesx/y.md", false},
		{"bare glob matches basename anywhere", []string{"*.test.ts"}, "src/deep/a.test.ts", true},
		{"bare glob mismatch", []string{"*.test.ts"}, "src/a.test.tsx", false},
		{"bare file name nested", []string{"package-lock.json"}, "pkg/a/package-lock.json", true},
		{"dot dir exact segment", []string{".git"}, ".github/workflows/ci.md", false},
		{"doublestar dir pattern at root", []string{"**/coverage/**"}, "coverage/f.md", true},
		{"doublestar dir pattern nested", []string{"**/coverage/**"}, "a/b/coverage/f.md", true},
		{"doublestar dir pattern near-miss", []string{"**/coverage/**"}, "a/coverages/f.md", false},
		{"doublestar suffix pattern", []string{"**/*.test.md"}, "a/b/c.test.md", true},
		{"path pattern is anchored", []string{"packages/dist"}, "x/packages/dist/f.md", false},
		{"path pattern matches subtree", []string{"packages/dist"}, "packages/dist/deep/f.md", true},
		{"path pattern exact", []string{"packages/dist"}, "packages/dist", true},
		{"leading slash anchors bare name", []string{"/dist"}, "dist/f.md", true},
		{"leading slash does not float", []string{"/dist"}, "a/dist/f.md", false},
		{"trailing slash means directory", []string{"githooks/"}, "githooks/hook.md", true},
		{"question mark single rune", []string{"?at.md"}, "cat.md", true},
		{"question mark needs a rune", []string{"?at.md"}, "at.md", false},
		{"character class", []string{"[ab]c.md"}, "bc.md", true},
		{"character class mismatch", []string{"[ab]c.md"}, "cc.md", false},
		{"negated class", []string{"[!a]c.md"}, "bc.md", true},
		{"negated class mismatch", []string{"[!a]c.md"}, "ac.md", false},
		{"class range", []string{"[a-c]x.md"}, "bx.md", true},
		{"empty patterns match nothing", nil, "anything.md", false},
		{"star within segment stays in segment", []string{"a*b/c.md"}, "axyb/c.md", true},
		{"star does not cross separator", []string{"a*b"}, "a/b", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewIgnoreMatcher(tc.patterns)
			if got := m.Matches(tc.path); got != tc.want {
				t.Fatalf("Matches(%q) with %v = %v, want %v", tc.path, tc.patterns, got, tc.want)
			}
		})
	}
}

func TestSharedConfigIgnorePaths(t *testing.T) {
	// The shipped shared config's ignorePaths, as this repo uses them.
	m := NewIgnoreMatcher([]string{
		"node_modules", "dist", "**/coverage/**", "coverage", ".git", "githooks",
		"package-lock.json", "**/*.test.ts", "**/*.test.mjs", "**/cspell.json",
		".gitignore", "tsconfig.json",
	})
	ignored := []string{
		"node_modules/pkg/README.md",
		"packages/checks/cspell/node_modules/typescript/README.md",
		"packages/shared/dist/index.md",
		"coverage/lcov-report/index.md",
		"packages/checks/cspell/src/runCspell.test.ts",
		"packages/shared/config/cspell.json",
		".gitignore",
		"githooks/README.md",
	}
	kept := []string{
		"README.md",
		"architecture/01-system-context.md",
		"packages/checks/cspell/README.md",
		"plans/01-go-rewrite.md",
	}
	for _, p := range ignored {
		if !m.Matches(p) {
			t.Errorf("expected %q to be ignored", p)
		}
	}
	for _, p := range kept {
		if m.Matches(p) {
			t.Errorf("expected %q to be kept", p)
		}
	}
}
