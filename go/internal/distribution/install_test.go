package distribution

import (
	"archive/tar"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestInstallAndCache(t *testing.T) {
	f := fixture(t)
	first := installed(t, f)
	second := installed(t, f)
	if first != second || f.downloads.Load() != 1 {
		t.Fatalf("cache missed: %s, %s, %d", first, second, f.downloads.Load())
	}
	data, err := os.ReadFile(filepath.Join(second, "fitness"))
	if err != nil || string(data) != testRunner {
		t.Fatalf("runner: %q %v", data, err)
	}
}

func TestRepairRetainsOldGeneration(t *testing.T) {
	f := fixture(t)
	old := installed(t, f)
	writeTestFile(t, filepath.Join(old, "fitness"), "damaged")
	fresh := installed(t, f)
	if fresh == old {
		t.Fatal("repair overwrote a generation already used by a process")
	}
	if _, err := os.Stat(old); err != nil {
		t.Fatal("old generation was removed", err)
	}
	if f.downloads.Load() != 1 {
		t.Fatal("verified archive fetched twice")
	}
}

func TestDamagedArchiveCache(t *testing.T) {
	f := fixture(t)
	bin := installed(t, f)
	writeTestFile(t, filepath.Join(filepath.Dir(bin), "archive.tar.gz"), "corrupt")
	installed(t, f)
	if f.downloads.Load() != 2 {
		t.Fatal("archive was not fetched again")
	}
}

func TestOfflineEmbeddedHashes(t *testing.T) {
	f := fixture(t)
	f.installer.Hashes = map[string]string{testPlatform: f.digest}
	installed(t, f)
	f.server.Close()
	installed(t, f)
}

func TestConcurrentInstall(t *testing.T) {
	f := fixture(t)
	var group sync.WaitGroup
	for range 4 {
		group.Go(func() {
			path, err := f.installer.Install(context.Background(), testVersion)
			if err != nil {
				t.Error(err)
				return
			}
			if _, err := os.Stat(filepath.Join(path, "fitness")); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
}

func TestCacheRejectsSymlink(t *testing.T) {
	f := fixture(t)
	bin := installed(t, f)
	path := filepath.Join(bin, "fitness")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/bin/false", path); err != nil {
		t.Fatal(err)
	}
	if fresh := installed(t, f); fresh == bin {
		t.Fatal("used a symlink as a cached binary")
	}
}

func TestLatestVersion(t *testing.T) {
	f := fixture(t)
	var log strings.Builder
	f.installer.Log = &log
	_, err := f.installer.Install(context.Background(), "latest")
	if err != nil || !strings.Contains(log.String(), "latest resolved to go/v"+testVersion) {
		t.Fatalf("latest: %s %v", log.String(), err)
	}
}

func TestDownloadError(t *testing.T) {
	f := fixture(t)
	f.status = http.StatusNotFound
	_, err := f.installer.Install(context.Background(), testVersion)
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("download status lost: %v", err)
	}
}

func TestMismatch(t *testing.T) {
	f := fixture(t)
	f.data = []byte("corrupt")
	_, err := f.installer.Install(context.Background(), testVersion)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("bad checksum: %v", err)
	}
}

func TestUnsafeArchive(t *testing.T) {
	cases := []testEntry{
		{"../escape", "bad", tar.TypeReg, 0755},
		{"/tmp/escape", "bad", tar.TypeReg, 0755},
		{testTop + "/fitness-check-link", "", tar.TypeSymlink, 0755},
		{testTop + "/fitness-check-hard", "", tar.TypeLink, 0755},
		{testTop + "/fitness", "duplicate", tar.TypeReg, 0755},
		{testTop + "/fitness-check-data", "bad", tar.TypeReg, 0600},
		{testTop + "/fitness/../../escape", "bad", tar.TypeReg, 0755},
	}
	for _, entry := range cases {
		t.Run(entry.name, func(t *testing.T) {
			f := fixture(t, entry)
			if _, err := f.installer.Install(context.Background(), testVersion); err == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
}

func TestCorruptArchive(t *testing.T) {
	f := fixture(t)
	f.data = []byte("not gzip")
	f.digest = contentHash(f.data)
	if _, err := f.installer.Install(context.Background(), testVersion); err == nil {
		t.Fatal("corrupt archive accepted")
	}
}

func TestMissingRunner(t *testing.T) {
	data := archiveBytes(t, []testEntry{{testTop + "/fitness-check-example", testRunner, tar.TypeReg, 0755}})
	if _, err := extractArchive(data, t.TempDir(), testTop); err == nil {
		t.Fatal("missing runner accepted")
	}
}

func TestDownloadSizeLimit(t *testing.T) {
	if _, err := readBounded(strings.NewReader("too long"), 3); err == nil {
		t.Fatal("size limit ignored")
	}
}

func TestTruncatedStream(t *testing.T) {
	if _, err := readBounded(failedReader{}, 1024); err == nil {
		t.Fatal("read failure ignored")
	}
}

type failedReader struct{}

func (failedReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
