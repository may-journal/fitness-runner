package requirements

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// pathWithout returns a PATH holding every real tool on the caller's PATH
// except names, since Linux runners keep Go and Node in /usr/bin.
func pathWithout(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		linkTools(dir, d, names)
	}
	return "PATH=" + dir
}

// killedVet returns a PATH whose go kills `go vet` as an out-of-memory kill
// would, and runs the real go otherwise.
func killedVet(t *testing.T) string {
	t.Helper()
	real, err := exec.LookPath("go")
	mustDo(t, err)
	dir := t.TempDir()
	write(t, dir, map[string]string{"go": "#!/bin/sh\n[ \"$1\" = vet ] && kill -9 $PPID\nexec \"" + real + "\" \"$@\"\n"})
	return "PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH")
}

// brokenTool returns a PATH whose name runs the shell script ahead of the real
// tools, so one tool misbehaves like a broken install.
func brokenTool(t *testing.T, name, script string) string {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, map[string]string{name: "#!/bin/sh\n" + script})
	return "PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH")
}

// crashingGo returns a PATH whose go prints message and exits 2 when its first
// argument is sub, else runs the real go.
func crashingGo(t *testing.T, sub, message string) string {
	t.Helper()
	real, err := exec.LookPath("go")
	mustDo(t, err)
	return brokenTool(t, "go", "[ \"$1\" = "+sub+" ] && { echo '"+message+"' >&2; exit 2; }\nexec \""+real+"\" \"$@\"\n")
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
