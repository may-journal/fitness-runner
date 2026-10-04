package release

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNativeConsumerWorkflow(t *testing.T) {
	c := testConfig(t)
	metadata := nativeFixture(t, c)
	t.Setenv("FITNESS_CACHE_DIR", t.TempDir())
	t.Setenv("FITNESS_TEST_TOOL_PATH", "")
	if err := c.Smoke(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	withDownload(t, c, metadata)
	if err := c.Smoke(context.Background(), true); err != nil {
		t.Fatal(err)
	}
}

func nativeFixture(t *testing.T, c Config) Metadata {
	t.Helper()
	version := "0.20261004.37"
	platform := runtime.GOOS + "-" + runtime.GOARCH
	directory := filepath.Join(c.Out, executable(runnerName, version, platform))
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	buildNative(t, filepath.Join(directory, runnerName), "", "./cmd/fitness")
	buildNative(t, filepath.Join(directory, "fitness-check-markdown-filename-kebab-case"), "", "./cmd/fitness-check-markdown-filename-kebab-case")
	buildNative(t, filepath.Join(directory, "fitness-check-markdown-links"), "", "./cmd/fitness-check-markdown-links")
	return finishNativeFixture(t, c, directory, version, platform)
}

func buildNative(t *testing.T, output, flags, target string) {
	t.Helper()
	_, source, _, _ := runtime.Caller(0)
	command := exec.Command("go", "build", "-ldflags", flags, "-o", output, target)
	command.Dir = filepath.Clean(filepath.Join(filepath.Dir(source), "../.."))
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("build %s: %v\n%s", target, err, data)
	}
}

func finishNativeFixture(t *testing.T, c Config, directory, version, platform string) Metadata {
	t.Helper()
	name := bundleName(version, platform)
	if err := archiveDirectory(filepath.Join(c.Out, name), directory); err != nil {
		t.Fatal(err)
	}
	asset, err := c.asset(name)
	if err != nil {
		t.Fatal(err)
	}
	metadata := Metadata{Version: version, Assets: []Asset{asset}}
	name = executable(installerName, version, platform)
	buildNative(t, filepath.Join(c.Out, name), "-X main.version="+version+" -X main.bundleHashes="+encodedHashes(metadata), "./cmd/fitness-install")
	return saveNativeFixture(t, c, metadata, name)
}

func saveNativeFixture(t *testing.T, c Config, metadata Metadata, name string) Metadata {
	t.Helper()
	asset, err := c.asset(name)
	if err != nil {
		t.Fatal(err)
	}
	metadata.Assets = append(metadata.Assets, asset)
	if err := c.saveMetadata(metadata); err != nil {
		t.Fatal(err)
	}
	return metadata
}

type downloadTransport func(*http.Request) (*http.Response, error)

func (f downloadTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func withDownload(t *testing.T, c Config, metadata Metadata) {
	t.Helper()
	original := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = original })
	http.DefaultClient = &http.Client{Transport: downloadTransport(func(request *http.Request) (*http.Response, error) {
		name := executable(installerName, metadata.Version, runtime.GOOS+"-"+runtime.GOARCH)
		expected := "https://github.com/may-journal/fitness-runner/releases/download/go/v" + metadata.Version + "/" + name
		if request.URL.String() != expected {
			t.Fatalf("download URL: %s", request.URL)
		}
		file, err := os.Open(filepath.Join(c.Out, name))
		return &http.Response{StatusCode: http.StatusOK, Body: file}, err
	})}
}

func TestDownloadResponseFailures(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
		_, err := readResponse(&http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("failure"))})
		requireError(t, err)
	}
}

func TestCorruptInstallerNeverWritten(t *testing.T) {
	c := testConfig(t)
	name := executable(installerName, "1.2.3", runtime.GOOS+"-"+runtime.GOARCH)
	testWrite(t, filepath.Join(c.Out, name), "corrupt executable")
	metadata := Metadata{Version: "1.2.3", Assets: []Asset{{Name: name, Hash: strings.Repeat("0", 64)}}}
	target := filepath.Join(t.TempDir(), "installer")
	requireError(t, c.prepareInstaller(context.Background(), metadata, target, false))
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("unverified executable written")
	}
}

func TestSmokeEnvironmentIsolation(t *testing.T) {
	t.Setenv("GIT_INDEX_FILE", "foreign-index")
	t.Setenv("FITNESS_CACHE_DIR", t.TempDir())
	t.Setenv("FITNESS_TEST_TOOL_PATH", t.TempDir())
	s, err := prepareSmoke(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range s.env {
		if strings.HasPrefix(entry, "GIT_") {
			t.Fatalf("inherited Git environment: %s", entry)
		}
	}
	assertNoGo(t, s)
}

func assertNoGo(t *testing.T, s smoke) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(s.tools, "go")); !os.IsNotExist(err) {
		t.Fatal("consumer has Go")
	}
	testWrite(t, filepath.Join(s.tools, "go"), "unexpected compiler")
	requireError(t, s.linkTools())
}

func TestArchiveRejectsDirectoriesAndLinks(t *testing.T) {
	for _, kind := range []string{"directory", "link"} {
		t.Run(kind, func(t *testing.T) { assertUnsafeEntry(t, kind) })
	}
}

func assertUnsafeEntry(t *testing.T, kind string) {
	t.Helper()
	directory := t.TempDir()
	path := filepath.Join(directory, "unsafe")
	var err error
	if kind == "directory" {
		err = os.Mkdir(path, 0700)
	} else {
		err = os.Symlink("missing", path)
	}
	if err != nil {
		t.Fatal(err)
	}
	requireError(t, archiveDirectory(filepath.Join(t.TempDir(), "bundle.tar.gz"), directory))
}
