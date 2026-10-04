package distribution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestActionInputs(t *testing.T) {
	env := map[string]string{"FITNESS_VERSION": "1.2.3", "FITNESS_CHECK": "example", "FITNESS_INSTALL_ONLY": "true"}
	options, err := actionInputs(Options{}, func(name string) string { return env[name] })
	want := Options{Version: "1.2.3", InstallOnly: true, Args: []string{"--check=example"}}
	if err != nil || !reflect.DeepEqual(options, want) {
		t.Fatalf("action options: %+v %v", options, err)
	}
}

func TestInvalidActionBoolean(t *testing.T) {
	_, err := actionInputs(Options{}, func(name string) string {
		if name == "FITNESS_INSTALL_ONLY" {
			return "maybe"
		}
		return ""
	})
	if err == nil {
		t.Fatal("invalid boolean accepted")
	}
}

func TestActionOutputFiles(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "outputs")
	t.Setenv("GITHUB_PATH", filepath.Join(root, "path"))
	t.Setenv("GITHUB_OUTPUT", output)
	if err := WriteActionFiles("/private/bin"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "bin=/private/bin\n" {
		t.Fatalf("output: %q %v", data, err)
	}
	if err := WriteActionFiles("bad\npath"); err == nil {
		t.Fatal("newline accepted")
	}
}

func TestGoHookConfiguration(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "installer")
	writeTestFile(t, binary, "compiled executable")
	hook := filepath.Join(root, "hooks", "pre-commit")
	options := Options{Version: testVersion, InstallHook: true, Args: []string{"--check=example", "a b"}}
	if err := writeHook(hook, binary, options); err != nil {
		t.Fatal(err)
	}
	assertHookConfig(t, hook, options.Args)
	if err := writeHook(hook, binary, options); err == nil {
		t.Fatal("existing hook replaced")
	}
}

func assertHookConfig(t *testing.T, hook string, args []string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(filepath.Dir(hook), hookConfigName))
	if err != nil {
		t.Fatal(err)
	}
	var options Options
	if err := json.Unmarshal(data, &options); err != nil {
		t.Fatal(err)
	}
	if options.InstallHook || !reflect.DeepEqual(options.Args, args) {
		t.Fatalf("hook config: %+v", options)
	}
}

func TestExternalActionInputs(t *testing.T) {
	env := map[string]string{"FITNESS_POLICY": "external", "FITNESS_CHECKS": "markdown-links,prose-budget"}
	options, err := actionInputs(Options{}, func(name string) string { return env[name] })
	want := []string{"--policy=external", "--checks=markdown-links,prose-budget"}
	if err != nil || !reflect.DeepEqual(options.Args, want) {
		t.Fatalf("options %+v: %v", options, err)
	}
	env["FITNESS_CHECK"] = "other"
	if _, err := actionInputs(Options{}, func(name string) string { return env[name] }); err == nil {
		t.Fatal("conflicting selections accepted")
	}
}
