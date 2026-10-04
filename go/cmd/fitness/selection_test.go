package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExternalConsumer(t *testing.T) {
	bins := buildChecks(t)
	root := t.TempDir()
	writeSelectionFile(t, root, "readme.md", "# Consumer\n")
	git := exec.Command("git", "init", "-q", root)
	git.Env = contractEnv()
	if output, err := git.CombinedOutput(); err != nil {
		t.Fatalf("git: %v %s", err, output)
	}
	add := exec.Command("git", "-C", root, "add", "readme.md")
	add.Env = contractEnv()
	if output, err := add.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v %s", err, output)
	}
	testConsumerSelections(t, bins, root)
}

func testConsumerSelections(t *testing.T, bins, root string) {
	t.Helper()
	config := `{"policy":"external","checks":["markdown-filename-kebab-case","markdown-links"],"disabledChecks":["markdown-links"]}`
	writeSelectionFile(t, root, ".fitnessrc.json", config)
	output := consumerRun(t, bins, root, 0, "--all")
	requireText(t, output, "policy=external checks=markdown-filename-kebab-case,markdown-links")
	requireText(t, output, "All 2 checks passed")
	output = consumerRun(t, bins, root, 0, "--checks=markdown-links", "--all")
	requireText(t, output, "All 1 checks passed")
	testMissingGo(t, bins, root)
	consumerRun(t, bins, root, 0, "--policy=org", "--check=markdown-links")
	assertConsumerUnchanged(t, root, config)
	testConsumerFailures(t, bins, root)
	testDefaultExternalSuite(t, bins, root)
}

func testConsumerFailures(t *testing.T, bins, root string) {
	t.Helper()
	for _, args := range [][]string{{"--checks="}, {"--checks=missing"}, {"--checks=markdown-links,markdown-links"}, {"--policy=invalid"}} {
		consumerRun(t, bins, root, 1, args...)
	}
	writeSelectionFile(t, root, ".fitnessrc.json", `{"policy":"external","checks":[]}`)
	consumerRun(t, bins, root, 1)
	consumerRun(t, bins, root, 0, "--checks=markdown-links")
	writeSelectionFile(t, root, ".fitnessrc.json", "{")
	consumerRun(t, bins, root, 1, "--checks=markdown-links")
}

func consumerRun(t *testing.T, bins, root string, status int, args ...string) string {
	t.Helper()
	cmd := exec.Command(filepath.Join(bins, "fitness"), args...)
	cmd.Dir, cmd.Env = root, consumerEnvironment(t)
	output, err := cmd.CombinedOutput()
	if cmd.ProcessState == nil {
		t.Fatalf("launch: %v", err)
	}
	if cmd.ProcessState.ExitCode() != status {
		t.Fatalf("%q: %v\n%s", args, err, output)
	}
	return string(output)
}

func consumerEnvironment(t *testing.T) []string {
	t.Helper()
	directory := t.TempDir()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(git, filepath.Join(directory, "git")); err != nil {
		t.Fatal(err)
	}
	return []string{"PATH=" + directory, "HOME=" + t.TempDir(), "LANG=C"}
}

func assertConsumerUnchanged(t *testing.T, root, config string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".fitnessrc.json"))
	if err != nil || string(data) != config {
		t.Fatalf("config changed: %s %v", data, err)
	}
	data, err = os.ReadFile(filepath.Join(root, "readme.md"))
	if err != nil || string(data) != "# Consumer\n" {
		t.Fatalf("file changed: %s %v", data, err)
	}
	assertNoConsumerHooks(t, root)
}

func assertNoConsumerHooks(t *testing.T, root string) {
	t.Helper()
	for _, name := range []string{"pre-commit", "pre-push", "commit-msg"} {
		if _, err := os.Stat(filepath.Join(root, ".git", "hooks", name)); !os.IsNotExist(err) {
			t.Fatalf("unexpected hook %s", name)
		}
	}
}

func writeSelectionFile(t *testing.T, root, name, value string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
}

func requireText(t *testing.T, output, want string) {
	t.Helper()
	if !strings.Contains(output, want) {
		t.Fatalf("missing %q in %s", want, output)
	}
}

func testMissingGo(t *testing.T, bins, root string) {
	t.Helper()
	writeSelectionFile(t, root, "go.mod", "module example.com/consumer\n\ngo 1.24\n")
	command := exec.Command("git", "-C", root, "add", "go.mod")
	command.Env = contractEnv()
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v %s", err, output)
	}
	output := consumerRun(t, bins, root, 1, "--checks=go-vet", "--all")
	requireText(t, output, "Go not installed")
}

func testDefaultExternalSuite(t *testing.T, bins, root string) {
	t.Helper()
	writeSelectionFile(t, root, ".fitnessrc.json", `{"policy":"external"}`)
	output := consumerRun(t, bins, root, 1, "--all")
	requireText(t, output, "policy=external checks="+strings.Join(allChecks, ","))
	for _, name := range allChecks {
		requireText(t, output, "→ "+name)
	}
	requireText(t, output, "Go not installed")
}
