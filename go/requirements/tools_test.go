package requirements

import (
	"os"
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
