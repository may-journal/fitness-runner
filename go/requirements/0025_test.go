package requirements

import "testing"

// gitignoreWhy runs `fitness-install -- gitignore-why` on happyRepo with
// the given .gitignore; an empty one means the repo has none.
func gitignoreWhy(t *testing.T, gitignore string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", map[string]string{".gitignore": gitignore})
	return fitness(t, repo, nil, "gitignore-why")
}

// gitignoreWhyNoComment is the tail of every gitignore-why failure.
const gitignoreWhyNoComment = "has no explanatory # comment on the line above"

func Test0025_1(t *testing.T) {
	out, code := gitignoreWhy(t, "# dependencies\nnode_modules/\n\n# build output\ndist\n\n# keep this tracked\n!dist/keep.js\n")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0025_2(t *testing.T) {
	out, code := gitignoreWhy(t, "node_modules/\n")
	sees(t, out, code, 1, `.gitignore:1: pattern "node_modules/" `+gitignoreWhyNoComment)
}

func Test0025_3(t *testing.T) {
	out, code := gitignoreWhy(t, "# deps\nnode_modules/\ndist\n")
	sees(t, out, code, 1, `.gitignore:3: pattern "dist" `+gitignoreWhyNoComment)
}

func Test0025_4(t *testing.T) {
	out, code := gitignoreWhy(t, "# deps\n\nnode_modules/\n")
	sees(t, out, code, 1, `.gitignore:3: pattern "node_modules/" `+gitignoreWhyNoComment)
}

func Test0025_5(t *testing.T) {
	out, code := gitignoreWhy(t, "#\nnode_modules/\n")
	sees(t, out, code, 1, `.gitignore:2: pattern "node_modules/" `+gitignoreWhyNoComment)
}

func Test0025_6(t *testing.T) {
	out, code := gitignoreWhy(t, "")
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "gitignore-why")
}
