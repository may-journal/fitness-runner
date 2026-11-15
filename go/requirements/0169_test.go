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
func install0169(t *testing.T, ref, event string, goAnswers, ghAnswers []response) (string, int, replays, string) {
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
		cmd.Env = append(cmd.Env, "GITHUB_EVENT_PATH="+filepath.Join(dir, "event.json"), "GITHUB_REPOSITORY=may-journal/example", "GITHUB_PATH="+pathFile)
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

func Test0169_1(t *testing.T) {
	t.Parallel()
	out, code, rp, pathFile := install0169(t, "deadbeef", `{"pull_request":{"number":7}}`, goAnswers0169("deadbeef"),
		[]response{{Match: []string{"pulls/7"}, Stdout: "cafe1234\n"}})
	sees(t, out, code, 0, "Pull request 7 merged", "its head deadbeef is gone", "merge commit cafe1234")
	if !rp.called("go", "install", "@cafe1234") {
		t.Errorf("go never built the merge commit: %q", rp.calls("go"))
	}
	if path, _ := os.ReadFile(pathFile); !strings.Contains(string(path), "/gopath/bin") {
		t.Errorf("the build is not on PATH: %q", path)
	}
}

func Test0169_2(t *testing.T) {
	t.Parallel()
	open := []response{{Match: []string{"pulls/7"}}}
	for event, answers := range map[string][]response{`{"pull_request":{"number":7}}`: open, `{}`: nil} {
		out, code, rp, _ := install0169(t, "deadbeef", event, goAnswers0169("deadbeef"), answers)
		sees(t, out, code, 1, "go install of deadbeef failed", "no other commit stands in")
		if rp.count("go", "install") != 1 {
			t.Errorf("go built more than the requested ref: %q", rp.calls("go"))
		}
	}
}
