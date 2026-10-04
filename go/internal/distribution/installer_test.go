// cspell:ignore Typeflag
package distribution

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const fixtureVersion = "1.2.3"
const fixtureRunner = "#!/bin/bash\nprintf '%s\\n' \"$@\" > \"$FITNESS_TEST_ARGS\"\nexit \"${FITNESS_TEST_EXIT:-0}\"\n"

type fixture struct {
	root, script, cache, tools, asset, digest string
	env                                       []string
}

type entry struct {
	name, body string
	kind       byte
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
}

func newFixture(t *testing.T, platform string, extra []entry) *fixture {
	t.Helper()
	root := t.TempDir()
	f := &fixture{root: root, cache: filepath.Join(root, "cache"), tools: filepath.Join(root, "tools")}
	f.script = filepath.Join(root, "fitness.sh")
	source, err := os.ReadFile("../../../fitness.sh")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, f.script, string(source))
	f.asset = "fitness-" + fixtureVersion + "-" + platform + ".tar.gz"
	f.archive(t, platform, extra)
	f.prepareTools(t, platform)
	f.env = append(os.Environ(), "PATH="+f.tools, "FITNESS_CACHE_DIR="+f.cache, "FITNESS_TEST_ROOT="+root, "FITNESS_TEST_ARGS="+filepath.Join(root, "args"))
	return f
}

func (f *fixture) archive(t *testing.T, platform string, extra []entry) {
	t.Helper()
	top := "fitness-" + fixtureVersion + "-" + platform + "/"
	entries := append([]entry{{top + "fitness", fixtureRunner, tar.TypeReg}, {top + "fitness-check-example", fixtureRunner, tar.TypeReg}}, extra...)
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		writeEntry(t, tw, e)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	f.digest = fmt.Sprintf("%x", sha256.Sum256(buf.Bytes()))
	if err := os.WriteFile(filepath.Join(f.root, f.asset), buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(f.root, "checksums.txt"), f.digest+"  "+f.asset+"\n")
}

func writeEntry(t *testing.T, tw *tar.Writer, e entry) {
	t.Helper()
	h := &tar.Header{Name: e.name, Mode: 0755, Typeflag: e.kind, Size: int64(len(e.body))}
	if e.kind == tar.TypeSymlink || e.kind == tar.TypeLink {
		h.Linkname = "../../escape"
		h.Size = 0
	}
	if err := tw.WriteHeader(h); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(e.body)); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) prepareTools(t *testing.T, platform string) {
	t.Helper()
	if err := os.Mkdir(f.tools, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range strings.Fields("bash awk mkdir mktemp rm rmdir sleep tar gzip sort uniq wc tr find mv shasum sha256sum") {
		linkTool(t, f.tools, name)
	}
	parts := strings.Split(platform, "-")
	system := map[string]string{"linux": "Linux", "darwin": "Darwin"}[parts[0]]
	arch := map[string]string{"amd64": "x86_64", "arm64": "arm64"}[parts[1]]
	writeFile(t, filepath.Join(f.tools, "uname"), "#!/bin/bash\nif [ \"$1\" = -s ]; then echo "+system+"; else echo "+arch+"; fi\n")
	writeFile(t, filepath.Join(f.tools, "curl"), fakeCurl)
}

func linkTool(t *testing.T, dir, name string) {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		return
	}
	if err := os.Symlink(path, filepath.Join(dir, name)); err != nil {
		t.Fatal(err)
	}
}

const fakeCurl = `#!/bin/bash
set -eu
[ "${FITNESS_TEST_CURL_FAIL:-0}" = 0 ] || exit 22
out=''
url=''
while [ "$#" -gt 0 ]; do
 case "$1" in
  -o) out=$2; shift 2 ;;
  --retry|-w) shift 2 ;;
  -*) shift ;;
  *) url=$1; shift ;;
 esac
done
if [[ "$url" == */latest ]]; then
 printf 'https://github.com/may-journal/fitness-runner/releases/tag/go/v1.2.3'
 exit 0
fi
name=${url##*/}
printf '%s\n' "$name" >> "$FITNESS_TEST_ROOT/downloads"
if [ -n "$out" ]; then
 /bin/cp "$FITNESS_TEST_ROOT/$name" "$out"
else
 /bin/cat "$FITNESS_TEST_ROOT/$name"
fi
exit "${FITNESS_TEST_CURL_EXIT:-0}"
`

func (f *fixture) run(args ...string) (string, string, error) {
	cmd := exec.Command("/bin/bash", append([]string{f.script}, args...)...)
	cmd.Env = f.env
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	err := cmd.Run()
	return strings.TrimSpace(out.String()), stderr.String(), err
}

func requireRun(t *testing.T, f *fixture, args ...string) string {
	t.Helper()
	out, stderr, err := f.run(args...)
	if err != nil {
		t.Fatalf("installer: %v\n%s", err, stderr)
	}
	return out
}

func TestPlatformsAndArguments(t *testing.T) {
	for _, platform := range []string{"linux-amd64", "linux-arm64", "darwin-amd64", "darwin-arm64"} {
		t.Run(platform, func(t *testing.T) {
			f := newFixture(t, platform, nil)
			requireRun(t, f, "--version", fixtureVersion, "--", "argument with spaces", "--check=example")
			got, err := os.ReadFile(filepath.Join(f.root, "args"))
			if err != nil || string(got) != "argument with spaces\n--check=example\n" {
				t.Fatalf("arguments: %q, %v", got, err)
			}
			if _, err := os.Stat(filepath.Join(f.tools, "go")); !os.IsNotExist(err) {
				t.Fatal("Go must be absent")
			}
		})
	}
}

func TestRejectsUnsafeArchives(t *testing.T) {
	top := "fitness-" + fixtureVersion + "-linux-amd64/"
	cases := []entry{{"../escape", "bad", tar.TypeReg}, {"/tmp/escape", "bad", tar.TypeReg}, {top + "fitness-check-link", "", tar.TypeSymlink}, {top + "fitness-check-hard", "", tar.TypeLink}, {top + "fitness", "duplicate", tar.TypeReg}, {top + "fitness-../escape", "bad", tar.TypeReg}}
	for _, e := range cases {
		t.Run(e.name, func(t *testing.T) {
			f := newFixture(t, "linux-amd64", []entry{e})
			_, _, err := f.run("--version", fixtureVersion)
			if err == nil {
				t.Fatal("unsafe archive accepted")
			}
			assertNotRun(t, f)
		})
	}
}

func assertNotRun(t *testing.T, f *fixture) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(f.root, "args")); !os.IsNotExist(err) {
		t.Fatal("runner was executed")
	}
}

func TestDownloadFailures(t *testing.T) {
	for _, variable := range []string{"FITNESS_TEST_CURL_FAIL=1", "FITNESS_TEST_CURL_EXIT=18"} {
		t.Run(variable, func(t *testing.T) {
			f := newFixture(t, "linux-amd64", nil)
			f.env = append(f.env, variable)
			_, _, err := f.run("--version", fixtureVersion)
			if err == nil {
				t.Fatal("download error was lost")
			}
			assertNotRun(t, f)
		})
	}
}

func TestChecksumMismatch(t *testing.T) {
	f := newFixture(t, "linux-amd64", nil)
	writeFile(t, filepath.Join(f.root, f.asset), "corrupt download")
	_, stderr, err := f.run("--version", fixtureVersion)
	if err == nil || !strings.Contains(stderr, "checksum mismatch") {
		t.Fatalf("%v: %s", err, stderr)
	}
	assertNotRun(t, f)
}

func TestCacheRepair(t *testing.T) {
	f := newFixture(t, "linux-amd64", nil)
	bin := requireRun(t, f, "--version", fixtureVersion, "--install-only")
	assertNotRun(t, f)
	writeFile(t, filepath.Join(bin, "fitness"), "broken")
	requireRun(t, f, "--version", fixtureVersion, "--", "ok")
	data, err := os.ReadFile(filepath.Join(f.root, "downloads"))
	if err != nil || strings.Count(string(data), f.asset) != 1 {
		t.Fatalf("cache fetched twice: %s, %v", data, err)
	}
}

func TestEmbeddedHashesAllowOfflineCache(t *testing.T) {
	f := newFixture(t, "darwin-arm64", nil)
	source, err := os.ReadFile(f.script)
	if err != nil {
		t.Fatal(err)
	}
	source = regexp.MustCompile(`(?m)^version=.*$`).ReplaceAll(source, []byte("version='"+fixtureVersion+"'"))
	source = bytes.Replace(source, []byte("bundle_hashes=''"), []byte("bundle_hashes='"+f.digest+"  "+f.asset+"'"), 1)
	writeFile(t, f.script, string(source))
	requireRun(t, f, "--install-only")
	f.env = append(f.env, "FITNESS_TEST_CURL_FAIL=1")
	requireRun(t, f, "--", "offline")
}

func TestVersionAndFailureStatus(t *testing.T) {
	f := newFixture(t, "linux-amd64", nil)
	_, stderr, err := f.run("--version", "latest", "--install-only")
	if err != nil || !strings.Contains(stderr, "version="+fixtureVersion) {
		t.Fatalf("latest: %v %s", err, stderr)
	}
	f.env = append(f.env, "FITNESS_TEST_EXIT=7")
	_, _, err = f.run("--version", fixtureVersion)
	if code, ok := err.(*exec.ExitError); !ok || code.ExitCode() != 7 {
		t.Fatalf("exit code lost: %v", err)
	}
}

func TestInvalidOptions(t *testing.T) {
	for _, args := range [][]string{{"--unknown"}, {"--version"}, {"--version", "../../main"}, {"--version", "main"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			f := newFixture(t, "linux-amd64", nil)
			_, _, err := f.run(args...)
			if err == nil {
				t.Fatal("invalid option accepted")
			}
		})
	}
}

func TestUnsupportedHost(t *testing.T) {
	f := newFixture(t, "linux-amd64", nil)
	writeFile(t, filepath.Join(f.tools, "uname"), "#!/bin/bash\necho unsupported\n")
	_, stderr, err := f.run("--version", fixtureVersion)
	if err == nil || !strings.Contains(stderr, "supported platforms") {
		t.Fatalf("%v: %s", err, stderr)
	}
	assertNotRun(t, f)
}

func TestDamagedArchiveCache(t *testing.T) {
	f := newFixture(t, "linux-amd64", nil)
	bin := requireRun(t, f, "--version", fixtureVersion, "--install-only")
	writeFile(t, bin+".tar.gz", "broken cache")
	requireRun(t, f, "--version", fixtureVersion)
	data, err := os.ReadFile(filepath.Join(f.root, "downloads"))
	if err != nil || strings.Count(string(data), f.asset) != 2 {
		t.Fatalf("cache was not repaired: %s %v", data, err)
	}
}

func TestCacheSymlinkRepair(t *testing.T) {
	f := newFixture(t, "linux-amd64", nil)
	bin := requireRun(t, f, "--version", fixtureVersion, "--install-only")
	runner := filepath.Join(bin, "fitness")
	if err := os.Remove(runner); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/bin/false", runner); err != nil {
		t.Fatal(err)
	}
	requireRun(t, f, "--version", fixtureVersion)
}
