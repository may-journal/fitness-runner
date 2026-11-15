package requirements

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// install0169 runs the fitness-install action's source build on ref, as an
// Actions run of event would, with go and gh answering as saved. It returns
// the output, the exit code, the stand-ins and the PATH file the script wrote.
func install0169(t *testing.T, repo, ref, event string, goAnswers, ghAnswers []response) (string, int, replays, string) {
	t.Helper()
	rp := standIn(t, map[string][]response{"go": goAnswers, "gh": ghAnswers})
	dir := t.TempDir()
	write(t, dir, map[string]string{"event.json": event})
	pathFile := filepath.Join(dir, "path")
	mustDo(t, os.WriteFile(pathFile, nil, 0o644))
	script, err := filepath.Abs("../../.github/actions/fitness-install/install-source.sh")
	mustDo(t, err)
	out, code := execute(t, func() *exec.Cmd {
		cmd := exec.Command(script, ref)
		cmd.Env = append(userEnv(), rp.env...)
		cmd.Env = append(cmd.Env, "GITHUB_EVENT_PATH="+filepath.Join(dir, "event.json"), "GITHUB_REPOSITORY="+repo, "GITHUB_PATH="+pathFile)
		return cmd
	})
	return out, code, rp, pathFile
}

// goAnswers0169 is go failing to fetch gone and building anything else.
func goAnswers0169(gone string) []response {
	return []response{
		{Match: []string{"install", "@" + gone}, Stderr: "unknown revision " + gone + "\n", Exit: 1},
		{Match: []string{"install"}},
		{Match: []string{"env", "GOPATH"}, Stdout: "/gopath\n"},
	}
}

// pull0169 is GitHub's answer for pull request 7 with head deadbeef.
func pull0169(merged string) []response {
	return []response{{Match: []string{"pulls/7"}, Stdout: `{"merged":` + merged + `,"head":{"sha":"deadbeef"},"merge_commit_sha":"cafe1234"}` + "\n"}}
}

func Test0169_1(t *testing.T) {
	t.Parallel()
	for _, event := range []string{`{"pull_request":{"number":7}}`, `{"issue":{"number":7}}`} {
		out, code, rp, pathFile := install0169(t, "may-journal/fitness-runner", "deadbeef", event, goAnswers0169("deadbeef"), pull0169("true"))
		sees(t, out, code, 0, "Pull request 7 merged", "its head deadbeef is gone", "merge commit cafe1234")
		if !rp.called("go", "install", "@cafe1234") {
			t.Errorf("go never built the merge commit: %q", rp.calls("go"))
		}
		if path, _ := os.ReadFile(pathFile); !strings.Contains(string(path), "/gopath/bin") {
			t.Errorf("the build is not on PATH: %q", path)
		}
	}
}

func Test0169_2(t *testing.T) {
	t.Parallel()
	cases := []struct {
		repo, ref, event string
		answers          []response
	}{
		{"may-journal/fitness-runner", "deadbeef", `{"pull_request":{"number":7}}`, pull0169("false")},
		{"may-journal/fitness-runner", "deadbeef", `{}`, nil},
		{"may-journal/fitness-runner", "abcdef12", `{"pull_request":{"number":7}}`, pull0169("true")},
		{"may-journal/other", "deadbeef", `{"pull_request":{"number":7}}`, pull0169("true")},
		{"may-journal/fitness-runner", "deadbeef", `{"pull_request":{"number":7}}`, []response{{Match: []string{"pulls/7"}, Exit: 1}}},
	}
	for _, c := range cases {
		out, code, rp, _ := install0169(t, c.repo, c.ref, c.event, goAnswers0169(c.ref), c.answers)
		sees(t, out, code, 1, "go install of "+c.ref+" failed", "no other commit stands in")
		if rp.count("go", "install") != 1 {
			t.Errorf("go built more than the requested ref: %q", rp.calls("go"))
		}
	}
}
