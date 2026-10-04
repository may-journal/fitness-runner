package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

func write(t *testing.T, root, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}
func TestOptions(t *testing.T) {
	root := t.TempDir()
	write(t, root, ".fitnessrc.json", `{"goTestCoverage":{"entries":["m/api"]}}`)
	options, err := parse(root, []string{"--audit"})
	if err != nil || !options.Audit || options.Entries[0] != "m/api" {
		t.Fatalf("%+v %v", options, err)
	}
}
func TestEntryOverride(t *testing.T) {
	root := t.TempDir()
	write(t, root, ".fitnessrc.json", `{"goTestCoverage":{"entries":["m/api"]}}`)
	options, err := parse(root, []string{"--entry=m/other"})
	if err != nil || options.Entries[0] != "m/other" {
		t.Fatalf("%+v %v", options, err)
	}
}
func TestInvalidOptions(t *testing.T) {
	cases := [][]string{{"--entry="}, {"--entry=a,,b"}, {"--entry=a,a"}, {"--entry= a"}, {"--unknown"}, {"unexpected"}}
	for _, args := range cases {
		if _, err := parse(t.TempDir(), args); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
}
func TestBadConfig(t *testing.T) {
	root := t.TempDir()
	for _, config := range []string{`{"goTestCoverage":{"entries":[]}}`, `{broken`} {
		write(t, root, ".fitnessrc.json", config)
		if result, _ := run(root, nil); result.Ok {
			t.Fatal("accepted bad config")
		}
	}
}
func TestRunNoModules(t *testing.T) {
	result, err := run(t.TempDir(), nil)
	if err != nil || !result.Ok || result.FilesChecked != 0 {
		t.Fatalf("%+v %v", result, err)
	}
}
func module(t *testing.T) string {
	root := t.TempDir()
	write(t, root, "go.mod", "module example.test/cli\n\ngo 1.24\n")
	write(t, root, "main.go", "package main\nfunc main() { println(42) }\n")
	return root
}
func TestRunEntryFailureAndSuccess(t *testing.T) {
	root := module(t)
	result, err := run(root, nil)
	if err != nil || result.Ok {
		t.Fatalf("%+v %v", result, err)
	}
	write(t, root, "main_test.go", "package main\nimport \"testing\"\nfunc TestEntry(t *testing.T) { main() }\n")
	result, err = run(root, nil)
	if err != nil || !result.Ok {
		t.Fatalf("%+v %v", result, err)
	}
}
func TestRunToolFailure(t *testing.T) {
	root := module(t)
	t.Setenv("PATH", t.TempDir())
	result, err := run(root, nil)
	if err != nil || result.Ok || !strings.Contains(strings.Join(result.Errors, ""), "Go not installed") {
		t.Fatalf("%+v %v", result, err)
	}
}
func TestCompiledProtocol(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "coverage")
	cmd := exec.Command("go", "build", "-cover", "-o", bin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s: %v", out, err)
	}
	root := module(t)
	invoke(t, bin, root, "--describe", 0)
	invoke(t, bin, root, "", 1)
	write(t, root, "main_test.go", "package main\nimport \"testing\"\nfunc TestEntry(t *testing.T) { main() }\n")
	invoke(t, bin, root, "", 0)
}
func invoke(t *testing.T, bin, root, flag string, status int) {
	t.Helper()
	args := []string{"--root", root}
	if flag != "" {
		args = append(args, flag)
	}
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), "GOCOVERDIR="+childCoverageDir(t))
	out, err := cmd.Output()
	if cmd.ProcessState.ExitCode() != status {
		t.Fatalf("%s: %v", out, err)
	}
	verifyProtocol(t, out, flag, status)
}
func childCoverageDir(t *testing.T) string {
	if dir := os.Getenv("FITNESS_GO_COVER_DIR"); dir != "" {
		return dir
	}
	return t.TempDir()
}
func verifyProtocol(t *testing.T, out []byte, flag string, status int) {
	t.Helper()
	if flag == "--describe" {
		verifyDescription(t, out)
		return
	}
	var result checkkit.Result
	if err := json.Unmarshal(out, &result); err != nil || result.Ok != (status == 0) {
		t.Fatalf("%s", out)
	}
}

func verifyDescription(t *testing.T, out []byte) {
	t.Helper()
	var desc checkkit.Describe
	if err := json.Unmarshal(out, &desc); err != nil || desc.Name != "go-test-coverage" {
		t.Fatalf("%s", out)
	}
}

func TestAuditMapOptionAndExport(t *testing.T) {
	root := module(t)
	write(t, root, "main_test.go", "package main\nimport \"testing\"\nfunc TestEntry(t *testing.T) { main() }\n")
	file := filepath.Join(t.TempDir(), "claims.json")
	result, err := run(root, []string{"--audit-map=" + file})
	if err != nil || !result.Ok {
		t.Fatalf("%+v %v", result, err)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatal(err)
	}
}
func TestAuditMapExportFailure(t *testing.T) {
	root := t.TempDir()
	result, err := run(root, []string{"--audit-map=" + filepath.Join(root, "claims.json")})
	if err != nil || result.Ok {
		t.Fatalf("%+v %v", result, err)
	}
}
