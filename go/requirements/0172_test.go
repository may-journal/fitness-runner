package requirements

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// script0172 is a check program whose --describe prints describe and whose
// run fails saying it ran, so the summary shows which checks ran.
func script0172(name, describe string) string {
	return "#!/bin/sh\n[ \"$1\" = --describe ] && { printf '%s\\n' '" + describe + "'; exit 0; }\n" +
		"echo '{\"ok\":false,\"errors\":[\"" + name + " ran\"],\"filesChecked\":1}'\nexit 1\n"
}

// run0172 runs the fitness in bundle in repo with stdin and args.
func run0172(t *testing.T, bundle, repo, stdin string, args ...string) (string, int) {
	t.Helper()
	return execute(t, func() *exec.Cmd {
		cmd := exec.Command(filepath.Join(bundle, "fitness"), args...)
		cmd.Dir, cmd.Env = repo, append(userEnv(), "FITNESS_CHECKS_DIR=")
		if stdin != "" {
			cmd.Stdin = strings.NewReader(stdin)
		}
		return cmd
	})
}

// absent fails when out names a check as having run.
func absent(t *testing.T, out string, ran ...string) {
	t.Helper()
	for _, name := range ran {
		if strings.Contains(out, name+" ran") {
			t.Errorf("%s ran, but the run should leave it out:\n%s", name, out)
		}
	}
}

func Test0172_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{"notes/plan.md": "# Plan\n", "src/main.go": "package main\n"})
	bundle := bundle0162(t)
	write(t, bundle, map[string]string{
		"fitness-check-go-files": script0172("go-files", `{"name":"go-files","files":"\\.go$"}`),
		"fitness-check-md-files": script0172("md-files", `{"name":"md-files","files":"\\.md$","exclude":"^CHANGELOG"}`),
	})
	out, code := run0172(t, bundle, repo, "", "--policy=external", "--checks=go-files,md-files", "--dir=notes")
	sees(t, out, code, 1, "md-files ran", "Skipped (no matching files): go-files")
	absent(t, out, "go-files")
	out, code = run0172(t, bundle, repo, "", "--policy=external", "--checks=go-files,md-files")
	sees(t, out, code, 1, "go-files ran", "md-files ran")
}

func Test0172_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	bundle := bundle0162(t)
	write(t, bundle, map[string]string{
		"fitness-check-node-files": script0172("node-files", `{"name":"node-files","needs":"(^|/)package\\.json$"}`),
		"fitness-check-any-files":  script0172("any-files", `{"name":"any-files"}`),
	})
	out, code := run0172(t, bundle, repo, "", "--policy=external", "--checks=node-files,any-files")
	sees(t, out, code, 1, "any-files ran", "Skipped (the repo tracks no file its needs pattern names): node-files")
	absent(t, out, "node-files")
}

func Test0172_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	bundle := bundle0162(t)
	write(t, bundle, map[string]string{
		"fitness-check-lookahead": script0172("lookahead", `{"name":"lookahead","files":"^(?!vendor/)"}`),
		"fitness-check-teleport":  script0172("teleport", `{"name":"teleport","requires":["teleport"]}`),
		"fitness-check-any-files": script0172("any-files", `{"name":"any-files"}`),
	})
	out, code := run0172(t, bundle, repo, "", "--policy=external", "--checks=lookahead,any-files")
	sees(t, out, code, 1, `check lookahead files pattern "^(?!vendor/)" does not compile`,
		"go install github.com/may-journal/fitness-checks/cmd/fitness-check-lookahead@latest")
	absent(t, out, "lookahead", "any-files")
	out, code = run0172(t, bundle, repo, "", "--policy=external", "--checks=teleport,any-files")
	sees(t, out, code, 1, "any-files ran", `Skipped (requires "teleport", which this fitness does not know; upgrade fitness): teleport`)
	absent(t, out, "teleport")
}

func Test0172_5(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{"notes/plan.md": "# Plan\n"})
	bundle := bundle0162(t)
	write(t, bundle, map[string]string{
		"fitness-check-md-files": "#!/bin/sh\n[ \"$1\" = --describe ] && { printf '%s\\n' '{\"name\":\"md-files\",\"files\":\"\\\\.md$\"}'; exit 0; }\n" +
			"echo '{\"ok\":true,\"errors\":[],\"filesChecked\":1}'\n",
		"fitness-check-go-files": script0172("go-files", `{"name":"go-files","files":"\\.go$"}`),
	})
	args := []string{"--policy=external", "--checks=md-files,go-files", "--dir=notes"}
	out, code := run0172(t, bundle, repo, "", args...)
	sees(t, out, code, 0, "All 2 checks passed, 1 skipped", "Skipped (no matching files): go-files")
	if !strings.Contains(row(out, "go-files"), "skipped") {
		t.Errorf("go-files must show as a skipped row:\n%s", out)
	}
	out, code = run0172(t, bundle, repo, "", append(args, "--format=json")...)
	sees(t, out, code, 0, `"status": "skipped"`, `"skipped": "no matching files"`)
	absent(t, out, "go-files")
}

func Test0172_4(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	bundle := bundle0162(t)
	write(t, bundle, map[string]string{
		"fitness-check-text-only": script0172("text-only", `{"name":"text-only","blob":true}`),
		"fitness-check-repo-wide": script0172("repo-wide", `{"name":"repo-wide","blob":true,"requires":["repo"]}`),
	})
	out, code := run0172(t, bundle, repo, "Some text.\n", "--blob", "-", "--checks=text-only,repo-wide")
	sees(t, out, code, 1, "text-only ran", "Skipped (requires repo): repo-wide")
	absent(t, out, "repo-wide")
}
