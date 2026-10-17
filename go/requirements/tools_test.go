package requirements

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// pathWithout returns a PATH setting that holds every real tool on the
// caller's PATH except names, so a run meets a machine without them. A
// plain /usr/bin:/bin is not enough: Linux runners keep Go and Node there.
func pathWithout(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		linkTools(dir, d, names)
	}
	return "PATH=" + dir
}

// killedVet returns a PATH setting whose go kills the check that runs
// `go vet`, as an out-of-memory kill would, and runs the real go otherwise.
// So go-vet ends without a verdict while every other check judges as usual.
func killedVet(t *testing.T) string {
	t.Helper()
	real, err := exec.LookPath("go")
	mustDo(t, err)
	dir := t.TempDir()
	write(t, dir, map[string]string{"go": "#!/bin/sh\n[ \"$1\" = vet ] && kill -9 $PPID\nexec \"" + real + "\" \"$@\"\n"})
	return "PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH")
}

// linkTools links each entry of src into dir, except names and entries an
// earlier PATH folder already supplied.
func linkTools(dir, src string, names []string) {
	entries, _ := os.ReadDir(src)
	for _, e := range entries {
		link := filepath.Join(dir, e.Name())
		if _, err := os.Lstat(link); err == nil || slices.Contains(names, e.Name()) {
			continue
		}
		_ = os.Symlink(filepath.Join(src, e.Name()), link)
	}
}
