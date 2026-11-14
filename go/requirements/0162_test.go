package requirements

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// script0162 is a check program whose --describe prints describe and whose
// run prints a failing result saying said.
func script0162(describe, said string) string {
	return "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '" + describe + "'; exit 0; }\n" +
		"echo '{\"ok\":false,\"errors\":[\"" + said + "\"],\"filesChecked\":1}'\nexit 1\n"
}

// bundle0162 places the installed runner alone in a fresh folder, as a clean
// install of the engine leaves it, and returns the folder.
func bundle0162(t *testing.T) string {
	t.Helper()
	bundle := t.TempDir()
	placeRunner(t, filepath.Join(installOnly(t), "fitness"), filepath.Join(bundle, "fitness"))
	return bundle
}

// run0162 runs the fitness in bundle in repo with env added.
func run0162(t *testing.T, bundle, repo string, env []string, args ...string) (string, int) {
	t.Helper()
	return execute(t, func() *exec.Cmd {
		cmd := exec.Command(filepath.Join(bundle, "fitness"), args...)
		cmd.Dir, cmd.Env = repo, append(userEnv(), env...)
		return cmd
	})
}

func Test0162_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	bundle, configured, onPath := bundle0162(t), t.TempDir(), t.TempDir()
	where := func(name, place string) string { return script0162(`{"name":"`+name+`"}`, name+" ran from "+place) }
	write(t, configured, map[string]string{"fitness-check-first": where("first", "the configured folder")})
	write(t, bundle, map[string]string{
		"fitness-check-first":  where("first", "beside fitness"),
		"fitness-check-second": where("second", "beside fitness"),
	})
	write(t, onPath, map[string]string{
		"fitness-check-first":  where("first", "PATH"),
		"fitness-check-second": where("second", "PATH"),
		"fitness-check-third":  where("third", "PATH"),
	})
	env := []string{"FITNESS_CHECKS_DIR=" + configured, "PATH=" + onPath + string(os.PathListSeparator) + os.Getenv("PATH")}
	out, code := run0162(t, bundle, repo, env, "--policy=external", "--checks=first,second,third")
	sees(t, out, code, 1, "first ran from the configured folder", "second ran from beside fitness", "third ran from PATH")
	describe, code := execute(t, func() *exec.Cmd {
		return exec.Command(filepath.Join(installOnly(t), "fitness-check-markdown-links"), "--describe")
	})
	sees(t, describe, code, 0, `"name":"markdown-links"`, `"protocol":1`)
}

func Test0162_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	bundle := bundle0162(t)
	write(t, bundle, map[string]string{
		"fitness-check-newer": script0162(`{"name":"newer","protocol":2}`, "newer ran"),
		"fitness-check-older": "#!/bin/sh\n[ \"$FITNESS_PROTOCOL\" = 1 ] || exit 2\n[ \"$1\" = --describe ] && { echo '{\"name\":\"older\"}'; exit 0; }\n" +
			"echo '{\"ok\":true,\"filesChecked\":1}'\n",
	})
	out, code := run0162(t, bundle, repo, nil, "--policy=external", "--checks=older,newer")
	sees(t, out, code, 1, "check newer speaks check protocol 2, but this fitness speaks protocol 1; reinstall it: go install github.com/may-journal/fitness-checks/cmd/fitness-check-newer@latest")
	if strings.Contains(out, "newer ran") {
		t.Errorf("a check of another protocol must not run:\n%s", out)
	}
	out, code = run0162(t, bundle, repo, nil, "older")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0162_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	bundle := bundle0162(t)
	write(t, bundle, map[string]string{"fitness-check-only": script0162(`{"name":"only"}`, "only ran")})
	out, code := run0162(t, bundle, repo, nil, "--no-builtins", "--policy=external", "--all")
	sees(t, out, code, 1, "only ran", "0 of 1 checks passed")
	for _, selected := range [][]string{{"markdown-links"}, {"check", "markdown-links"}} {
		out, code = run0162(t, bundle, repo, nil, append([]string{"--no-builtins"}, selected...)...)
		sees(t, out, code, 1, "check markdown-links is not installed, and --no-builtins turns the built-in checks off; install it: go install github.com/may-journal/fitness-checks/cmd/fitness-check-markdown-links@latest")
	}
}

func Test0162_4(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	bundle := bundle0162(t)
	write(t, bundle, map[string]string{"fitness-check-local-branch": script0162(`{"name":"local-branch","scope":"github"}`, "installed local-branch ran")})
	out, code := run0162(t, bundle, repo, nil, "local-branch")
	sees(t, out, code, 0, "skipped")
	if strings.Contains(out, "installed local-branch ran") {
		t.Errorf("an installed local-branch cannot judge, so fitness must not run it:\n%s", out)
	}
}
