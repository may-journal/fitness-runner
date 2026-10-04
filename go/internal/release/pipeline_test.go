package release

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testWrite(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}

func testConfig(t *testing.T) Config {
	t.Helper()
	c, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	testWrite(t, filepath.Join(c.Root, "CHANGELOG.md"), "# Changelog\n\n### 2026.10.04.0037\n\n- Current release\n\n### 2026.10.03.1200\n\n- Previous release\n")
	testWrite(t, filepath.Join(c.Out, "stale"), "old output")
	return c
}

func requireError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestBuildReleaseWithRealCompiler(t *testing.T) {
	c := testConfig(t)
	testWrite(t, filepath.Join(c.Root, "go", "go.mod"), "module example.com/fixture\n\ngo 1.24\n")
	source := "package main\nvar version, bundleHashes string\nfunc main() { println(version, bundleHashes) }\n"
	for _, name := range []string{runnerName, installerName, verifierName, "fitness-example"} {
		testWrite(t, filepath.Join(c.Root, "go", "cmd", name, "main.go"), source)
	}
	if err := c.Build(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertRelease(t, c)
}

func assertRelease(t *testing.T, c Config) {
	t.Helper()
	metadata, err := c.Metadata()
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Version != "0.20261004.37" || len(metadata.Assets) != 8 {
		t.Fatalf("metadata: %+v", metadata)
	}
	if _, err := os.Stat(filepath.Join(c.Out, "stale")); !os.IsNotExist(err) {
		t.Fatal("stale output survived build")
	}
	assertAssets(t, c, metadata)
}

func assertAssets(t *testing.T, c Config, metadata Metadata) {
	t.Helper()
	for _, asset := range metadata.Assets {
		actual, err := c.asset(asset.Name)
		if err != nil || actual != asset {
			t.Fatalf("asset %s: %+v %v", asset.Name, actual, err)
		}
	}
	checksums, err := os.ReadFile(filepath.Join(c.Out, "checksums.txt"))
	if err != nil {
		t.Fatal(err)
	}
	assertChecksums(t, string(checksums), metadata)
	assertEmbeddedVersion(t, c, metadata)
}

func assertChecksums(t *testing.T, text string, metadata Metadata) {
	t.Helper()
	if len(strings.Split(strings.TrimSpace(text), "\n")) != 8 {
		t.Fatalf("checksums: %s", text)
	}
	for _, asset := range metadata.Assets {
		if !strings.Contains(text, asset.Hash+"  "+asset.Name+"\n") {
			t.Fatalf("missing checksum for %s", asset.Name)
		}
	}
}

func assertEmbeddedVersion(t *testing.T, c Config, metadata Metadata) {
	t.Helper()
	name := executable(installerName, metadata.Version, runtime.GOOS+"-"+runtime.GOARCH)
	output, err := exec.Command(filepath.Join(c.Out, name)).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), metadata.Version) {
		t.Fatalf("version not embedded: %s", output)
	}
	for _, asset := range metadata.Assets[:4] {
		if !strings.Contains(string(output), asset.Hash) {
			t.Fatalf("hash not embedded: %s", asset.Name)
		}
	}
}

func TestTagAndMetadataFailures(t *testing.T) {
	c := testConfig(t)
	if err := c.VerifyTag("go/v0.20261004.37"); err != nil {
		t.Fatal(err)
	}
	requireError(t, c.VerifyTag("go/v0.20261004.38"))
	_, err := c.Metadata()
	requireError(t, err)
	testWrite(t, filepath.Join(c.Out, metadataName), "invalid JSON")
	_, err = c.Metadata()
	requireError(t, err)
}

func TestBuildRejectsMissingSources(t *testing.T) {
	c := testConfig(t)
	requireError(t, c.Build(context.Background()))
	if _, err := c.Metadata(); err == nil {
		t.Fatal("failed build published metadata")
	}
	requireError(t, c.VerifyTag("wrong"))
}

func TestPublishRejectsWrongTag(t *testing.T) {
	c := testConfig(t)
	if err := c.saveMetadata(Metadata{Version: "0.20261004.37"}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_REF_NAME", "go/v0.20261004.38")
	requireError(t, c.Publish(context.Background()))
	if _, err := os.Stat(filepath.Join(c.Out, "notes.md")); !os.IsNotExist(err) {
		t.Fatal("publish continued after tag mismatch")
	}
}

func TestMain(m *testing.M) {
	if os.Getenv("FITNESS_RELEASE_TEST_PROCESS") == "gh" {
		data, _ := json.Marshal(os.Args[1:])
		if err := os.WriteFile(os.Getenv("FITNESS_RELEASE_TEST_ARGS"), data, 0600); err != nil {
			os.Exit(2)
		}
		if os.Getenv("FITNESS_RELEASE_TEST_FAIL") == "1" {
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestPublishArgumentsAndFailure(t *testing.T) {
	c := testConfig(t)
	metadata := Metadata{Version: "0.20261004.37", Assets: []Asset{{Name: "bundle.tar.gz", Hash: "digest"}}}
	if err := c.saveMetadata(metadata); err != nil {
		t.Fatal(err)
	}
	fakePublisher(t, c)
	t.Setenv("GITHUB_REF_NAME", "go/v0.20261004.37")
	if err := c.Publish(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertPublish(t, c)
	t.Setenv("FITNESS_RELEASE_TEST_FAIL", "1")
	requireError(t, c.Publish(context.Background()))
}

func fakePublisher(t *testing.T, c Config) {
	t.Helper()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err := os.Symlink(binary, filepath.Join(directory, "gh")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	t.Setenv("FITNESS_RELEASE_TEST_PROCESS", "gh")
	t.Setenv("FITNESS_RELEASE_TEST_ARGS", filepath.Join(c.Root, "arguments.json"))
}

func assertPublish(t *testing.T, c Config) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(c.Root, "arguments.json"))
	if err != nil {
		t.Fatal(err)
	}
	var args []string
	if err := json.Unmarshal(data, &args); err != nil {
		t.Fatal(err)
	}
	expected := []string{"release", "create", "go/v0.20261004.37", filepath.Join(c.Out, "bundle.tar.gz"), filepath.Join(c.Out, "checksums.txt"), "--verify-tag", "--title", "fitness v0.20261004.37", "--notes-file", filepath.Join(c.Out, "notes.md")}
	if strings.Join(args, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("publish arguments: %q", args)
	}
	assertNotes(t, c)
}

func assertNotes(t *testing.T, c Config) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(c.Out, "notes.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Current release") || strings.Contains(string(data), "Previous release") {
		t.Fatalf("notes: %s", data)
	}
}
