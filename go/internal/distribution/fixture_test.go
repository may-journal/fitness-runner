// cspell:ignore Typeflag
package distribution

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

const testVersion = "1.2.3"
const testPlatform = "linux-amd64"
const testTop = "fitness-" + testVersion + "-" + testPlatform
const testAsset = testTop + ".tar.gz"
const testRunner = "#!/bin/sh\nprintf '%s\\n' \"$@\"\nexit 0\n"

type testEntry struct {
	name, body string
	kind       byte
	mode       int64
}
type testFixture struct {
	installer *Installer
	data      []byte
	digest    string
	status    int
	downloads atomic.Int32
	server    *httptest.Server
}

func fixture(t *testing.T, extra ...testEntry) *testFixture {
	t.Helper()
	f := &testFixture{status: http.StatusOK}
	entries := append([]testEntry{{testTop + "/fitness", testRunner, tar.TypeReg, 0755}, {testTop + "/fitness-check-example", testRunner, tar.TypeReg, 0755}}, extra...)
	f.data = archiveBytes(t, entries)
	f.digest = contentHash(f.data)
	f.server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.server.Close)
	f.installer = &Installer{Version: testVersion, Platform: testPlatform, Cache: t.TempDir(), ReleaseURL: f.server.URL, LatestURL: f.server.URL + "/latest", Client: f.server.Client(), Log: io.Discard}
	return f
}

func (f *testFixture) serve(w http.ResponseWriter, r *http.Request) {
	if f.status != http.StatusOK {
		w.WriteHeader(f.status)
		return
	}
	switch filepath.Base(r.URL.Path) {
	case "latest":
		fmt.Fprintf(w, `{"tag_name":"go/v%s"}`, testVersion)
	case checksumName:
		fmt.Fprintf(w, "%s  %s\n", f.digest, testAsset)
	default:
		f.downloads.Add(1)
		w.Write(f.data)
	}
}

func archiveBytes(t *testing.T, entries []testEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	writer := tar.NewWriter(gz)
	for _, entry := range entries {
		writeEntry(t, writer, entry)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func writeEntry(t *testing.T, writer *tar.Writer, entry testEntry) {
	t.Helper()
	header := &tar.Header{Name: entry.name, Mode: entry.mode, Typeflag: entry.kind, Size: int64(len(entry.body))}
	if entry.kind == tar.TypeSymlink || entry.kind == tar.TypeLink {
		header.Linkname = "../../escape"
	}
	if err := writer.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte(entry.body)); err != nil {
		t.Fatal(err)
	}
}

func installed(t *testing.T, f *testFixture) string {
	t.Helper()
	path, err := f.installer.Install(context.Background(), testVersion)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func writeTestFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0700); err != nil {
		t.Fatal(err)
	}
}
