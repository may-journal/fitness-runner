package requirements

import "testing"

// unchangedTestFails checks ownership when staged and PR diffs leave tests alone.
func unchangedTestFails(t *testing.T, files map[string]string, want string) {
	t.Helper()
	for _, path := range []string{"README.md", "docs/requirements/0001-adds-numbers.md"} {
		for _, pr := range []bool{false, true} {
			repo := example(t, "happyRepo", files)
			git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
			write(t, repo, map[string]string{path: files[path] + "\n"})
			git(t, repo, "add", path)
			var env []string
			if pr {
				commit(t, repo, nil, "docs: change prose")
				env = []string{"GITHUB_ACTIONS=true", "GITHUB_BASE_REF=main"}
			}
			out, code := fitness(t, repo, env, "requirements")
			sees(t, out, code, 1, want)
		}
	}
}
