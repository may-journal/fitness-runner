package walkfs

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// TrackedFilesEnv names the variable through which the runner hands every
// check the file it wrote with ShareTracked, so a run lists tracked files
// once instead of once per check.
const TrackedFilesEnv = "FITNESS_TRACKED_FILES"

// trackedMemo holds each root's tracked-file list for the life of the
// process, so a check that walks the repo for several file types asks git
// once.
var trackedMemo = struct {
	sync.Mutex
	lists map[string]trackedList
}{lists: map[string]trackedList{}}

// trackedList is one root's `git ls-files` answer; ok is false outside a
// git work tree.
type trackedList struct {
	files []string
	ok    bool
}

// trackedFiles returns root's tracked paths, slash-separated and relative to
// root, as `git ls-files --cached` prints them, including any deleted from
// the work tree. ok is false when root is not inside a git work tree or git
// is missing. A list the runner shared for root is used first; otherwise git
// is asked once per root for the life of the process.
func trackedFiles(root string) (files []string, ok bool) {
	key := filepath.Clean(root)
	trackedMemo.Lock()
	defer trackedMemo.Unlock()
	if l, found := trackedMemo.lists[key]; found {
		return l.files, l.ok
	}
	l, shared := sharedTracked(key)
	if !shared {
		l = listTracked(key)
	}
	trackedMemo.lists[key] = l
	return l.files, l.ok
}

// listTracked asks git for root's tracked paths.
func listTracked(root string) trackedList {
	out, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached").Output()
	if err != nil {
		return trackedList{}
	}
	return trackedList{files: splitNUL(out), ok: true}
}

// ShareTracked lists root's tracked files once and writes them to a
// temporary file for the checks of one run, returning the TrackedFilesEnv
// entry that points at it and a cleanup that removes it. Outside a git work
// tree, or when the file cannot be written, it returns no entry, and each
// check lists its own files as it does when run alone.
func ShareTracked(root string) (env []string, cleanup func()) {
	cleanup = func() {}
	l := listTracked(root)
	if !l.ok {
		return nil, cleanup
	}
	f, err := os.CreateTemp("", "fitness-tracked-*")
	if err != nil {
		return nil, cleanup
	}
	remove := func() { _ = os.Remove(f.Name()) }
	_, writeErr := f.Write(encodeTracked(filepath.Clean(root), l.files))
	if closeErr := f.Close(); writeErr != nil || closeErr != nil {
		remove()
		return nil, cleanup
	}
	return []string{TrackedFilesEnv + "=" + f.Name()}, remove
}

// encodeTracked is the shared file's content: the root it lists, then each
// tracked path, every entry NUL-terminated.
func encodeTracked(root string, files []string) []byte {
	var b bytes.Buffer
	b.WriteString(root)
	b.WriteByte(0)
	for _, f := range files {
		b.WriteString(f)
		b.WriteByte(0)
	}
	return b.Bytes()
}

// sharedTracked reads the runner's list when it names root. A list for
// another root, such as one inherited by a test's nested run, is ignored,
// as is a missing or unreadable file.
func sharedTracked(root string) (trackedList, bool) {
	path := os.Getenv(TrackedFilesEnv)
	if path == "" {
		return trackedList{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return trackedList{}, false
	}
	listed, rest, found := bytes.Cut(data, []byte{0})
	if !found || string(listed) != root {
		return trackedList{}, false
	}
	return trackedList{files: splitNUL(rest), ok: true}, true
}

// splitNUL splits NUL-terminated entries, dropping empty ones.
func splitNUL(out []byte) []string {
	var files []string
	for _, rel := range bytes.Split(out, []byte{0}) {
		if len(rel) > 0 {
			files = append(files, string(rel))
		}
	}
	return files
}
