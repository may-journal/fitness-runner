package release

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionAndNotes(t *testing.T) {
	changelog := "# Changelog\n\n### 2026.10.04.0037\n\n- Current\n\n### 2026.10.03.2210\n\n- Old\n"
	version, err := Version([]byte(changelog))
	if err != nil || version != "0.20261004.37" {
		t.Fatalf("version=%s %v", version, err)
	}
	if strings.Contains(Notes(changelog), "Old") {
		t.Fatal("release notes included older changes")
	}
	if _, err := Version([]byte("missing")); err == nil {
		t.Fatal("missing heading accepted")
	}
}

func TestArchiveContainsOnlyBinaries(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "fitness-1.2.3-linux-amd64")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, runnerName), []byte("binary"), 0700); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "bundle.tar.gz")
	if err := archiveDirectory(archive, directory); err != nil {
		t.Fatal(err)
	}
	assertArchive(t, archive, filepath.Base(directory)+"/fitness")
}

func assertArchive(t *testing.T, path, want string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	assertEntry(t, tar.NewReader(reader), want)
}

func assertEntry(t *testing.T, reader *tar.Reader, want string) {
	t.Helper()
	header, err := reader.Next()
	if err != nil {
		t.Fatal(err)
	}
	assertHeader(t, header, want)
	data, err := io.ReadAll(reader)
	if err != nil || string(data) != "binary" {
		t.Fatalf("archive data: %s %v", data, err)
	}
}

func assertHeader(t *testing.T, header *tar.Header, want string) {
	t.Helper()
	if header.Name != want || header.Mode != 0755 {
		t.Fatalf("header: %+v", header)
	}
}

func TestAssetHashValidation(t *testing.T) {
	metadata := Metadata{Assets: []Asset{{Name: "installer", Hash: strings.Repeat("0", 64)}}}
	for _, name := range []string{"missing", "installer"} {
		if err := verifyAsset(metadata, name, []byte("corrupt")); err == nil {
			t.Fatal("bad asset accepted")
		}
	}
}
