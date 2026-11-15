package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// unpinnedPrePush is the pre-push hook as synced before it set
// FITNESS_WITH_CHECKS, which hands the engine no checks.
const unpinnedPrePush = "#!/bin/sh\nexec \"${FITNESS_INSTALL_DIR:-$HOME/.local/share/fitness}/fitness-install\" --version latest -- hook pre-push\n"

// push0170 pushes to a PR with a bold body from a repo pinning every check,
// none installed. It runs hook, else the `fitness init` one.
func push0170(t *testing.T, hook string) (string, int) {
	t.Helper()
	repo := branchRepo(t, map[string]string{".fitnessrc.json": `{"checkVersions": {"*": "v1"}}`})
	const title = "test(app): add one file"
	pr := branchAnswer(t, repo, title, pr0014(11)+"\nSome **bold** text.\n")
	rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issue0014(t, 11, planBody("- [x] Write the tests."), "Plan"),
		planView(t, 11, planBody("- [x] Write the tests."), []string{"Plan"}, "I approve"))})
	out, code := fitness(t, repo, nil, "init")
	sees(t, out, code, 0, "installed hooks")
	if hook != "" {
		mustDo(t, os.WriteFile(filepath.Join(repo, ".githooks", "pre-push"), []byte(hook), 0o755))
	}
	bare := t.TempDir()
	git(t, bare, "init", "-q", "--bare")
	git(t, repo, "remote", "set-url", "--push", "origin", bare)
	env := append(pushEnv(installerEnv(t), rp), "FITNESS_CHECKS_DIR=")
	out, code = user(t, repo, env, "commit", "--allow-empty", "-m", title)
	sees(t, out, code, 0, title)
	return user(t, repo, env, "push", "origin", "work")
}

func Test0170_1(t *testing.T) {
	t.Parallel()
	for name, hook := range map[string]string{"synced": "", "unpinned": unpinnedPrePush} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			out, code := push0170(t, hook)
			sees(t, out, code, 1, "pr-check #1", "**bold**")
			if strings.Contains(out, "no result from the check") {
				t.Errorf("the %s pre-push hook ran the PR body checks without installing them:\n%s", name, out)
			}
		})
	}
}
