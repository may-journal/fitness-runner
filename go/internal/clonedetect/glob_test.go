package clonedetect

import "testing"

func TestMatchGlob(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"**/*.md", "README.md", true},
		{"**/*.md", "docs/deep/nested/a.md", true},
		{"**/*.md", "a.mdx", false},
		{"**/*.json", "package.json", true},
		{"**/*.json", ".eslintrc.json", true}, // dot:true semantics
		{"**/*.json", "src/tsconfig.json", true},
		{"**/*.lock", "yarn.lock", true},
		{"**/*.lock", "sub/dir/Cargo.lock", true},
		{"**/*.lock", "lock.txt", false},
		{"**/*.test.*", "src/foo.test.ts", true},
		{"**/*.test.*", "foo.test.js", true},
		{"**/*.test.*", "protest.ts", false},
		{"**/*.test.*", "src/foo.test", false},
		{"**/*.spec.*", "deep/a/b/thing.spec.tsx", true},
		{"**/*_test.go", "main_test.go", true},
		{"**/*_test.go", "go/cmd/x/main_test.go", true},
		{"**/*_test.go", "go/cmd/x/main_test.gone", false},
		{"**/*_test.go", "go/cmd/x/main-test.go", false},
		{"**", "anything/at/all", true},
		{"*.lock", "yarn.lock", true},
		{"*.lock", "sub/yarn.lock", false}, // no ** means single segment
		{"a/?.go", "a/b.go", true},
		{"a/?.go", "a/bc.go", false},
		{"**/dist/**", "pkg/dist/index.js", true},
		{"**/dist/**", "pkg/src/index.js", false},
	}
	for _, tc := range cases {
		t.Run(tc.pattern+" vs "+tc.path, func(t *testing.T) {
			if got := MatchGlob(tc.pattern, tc.path); got != tc.want {
				t.Fatalf("MatchGlob(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
			}
		})
	}
}
