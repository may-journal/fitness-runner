package walkfs

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// TrackedFilesEnv names the variable through which the runner hands every
// check the file it wrote with ShareTracked, so a run lists tracked files
// once instead of once per check.
const TrackedFilesEnv = "FITNESS_TRACKED_FILES"

// trackedMemo holds each root's tracked-file list for the life of the
// process, so a check that walks the repo for several file types asks git
// once. The lock guards only the map; each root fills its own entry, so a
// walk of one root never waits on git for another.
var trackedMemo = struct {
	sync.Mutex
	roots map[string]*trackedEntry
}{roots: map[string]*trackedEntry{}}

// trackedEntry is one root's list, filled once.
type trackedEntry struct {
	once sync.Once
	list trackedList
}

// trackedList is one root's `git ls-files` answer; ok is false outside a
// git work tree.
type trackedList struct {
	files []string
	ok    bool
}

// trackedFiles returns root's tracked paths, slash-separated and relative to
// root, as `git ls-files --cached` prints them, including any deleted from
// the work tree. ok is false when root is not inside a git work tree or git
// is missing. A current list the runner shared for root is used first;
// otherwise git is asked once per root for the life of the process.
func trackedFiles(root string) (files []string, ok bool) {
	key := filepath.Clean(root)
	trackedMemo.Lock()
	entry := trackedMemo.roots[key]
	if entry == nil {
		entry = &trackedEntry{}
		trackedMemo.roots[key] = entry
	}
	trackedMemo.Unlock()
	entry.once.Do(func() {
		var shared bool
		if entry.list, shared = sharedTracked(key); !shared {
			entry.list, _ = listTracked(key)
		}
	})
	return entry.list.files, entry.list.ok
}

// listTracked asks git for root's tracked paths, returning git's error when
// it cannot list them.
func listTracked(root string) (trackedList, error) {
	files, err := gitx.Repo(root).TrackedFiles()
	if err != nil {
		return trackedList{}, err
	}
	return trackedList{files: files, ok: true}, nil
}

// ShareTracked lists root's tracked files once and writes them to a
// temporary file for the checks of one run, returning the TrackedFilesEnv
// entry that points at it and a cleanup that removes it. Outside a git work
// tree, or when the index or the file cannot be read or written, it returns
// no entry, and each check lists its own files as it does when run alone. A
// git call stopped at its deadline returns that gitx.StalledError instead.
func ShareTracked(root string) (env []string, cleanup func(), err error) {
	cleanup = func() {}
	content, err := trackedSnapshot(filepath.Clean(root))
	if content == nil {
		return nil, cleanup, err
	}
	f, err := os.CreateTemp("", "fitness-tracked-*")
	if err != nil {
		return nil, cleanup, nil
	}
	remove := func() { _ = os.Remove(f.Name()) }
	_, writeErr := f.Write(content)
	if closeErr := f.Close(); writeErr != nil || closeErr != nil {
		remove()
		return nil, cleanup, nil
	}
	return []string{TrackedFilesEnv + "=" + f.Name()}, remove, nil
}

// trackedSnapshot is the shared file's content for root, or nil when git or
// the index cannot be read. The error is set only for a stalled git call.
func trackedSnapshot(root string) ([]byte, error) {
	index, err := gitx.Repo(root).GitPath("index")
	if err != nil {
		return nil, stalledOnly(err)
	}
	// The stamp is read before the listing, so an index rewritten in
	// between only makes checks list their own files.
	stamp, ok := indexStamp(index)
	l, err := listTracked(root)
	if err != nil || !ok {
		return nil, stalledOnly(err)
	}
	return encodeTracked([]string{root, index, stamp}, l.files), nil
}

// stalledOnly keeps err when it is a git call stopped at its deadline, and
// drops any other failure, which only means checks list their own files.
func stalledOnly(err error) error {
	if gitx.Stalled(err) {
		return err
	}
	return nil
}

// indexStamp names the index's current version by its inode, modification
// time in nanoseconds, and size. Git writes a new index file and renames it
// into place, so even a same-size rewrite within one mtime tick gets a new
// inode and a new stamp.
func indexStamp(index string) (string, bool) {
	info, err := os.Stat(index)
	if err != nil {
		return "", false
	}
	var inode uint64
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		inode = st.Ino
	}
	return fmt.Sprintf("%d:%d:%d", inode, info.ModTime().UnixNano(), info.Size()), true
}

// encodeTracked is the shared file's content: the header entries (root,
// index path, index stamp), then each tracked path, every entry
// NUL-terminated.
func encodeTracked(header, files []string) []byte {
	var b bytes.Buffer
	for _, entry := range append(header, files...) {
		b.WriteString(entry)
		b.WriteByte(0)
	}
	return b.Bytes()
}

// sharedTracked reads the runner's list when it names root and the index
// has not changed since. A list for another root, such as one inherited by
// a test's nested run, is ignored, as is a stale, missing, or malformed one.
func sharedTracked(root string) (trackedList, bool) {
	path := os.Getenv(TrackedFilesEnv)
	if path == "" {
		return trackedList{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return trackedList{}, false
	}
	return parseShared(root, data)
}

// parseShared decodes the shared file's content, accepting it only when it
// lists root and its index stamp is still current.
func parseShared(root string, data []byte) (trackedList, bool) {
	entries := bytes.SplitN(data, []byte{0}, 4)
	if len(entries) < 4 || string(entries[0]) != root {
		return trackedList{}, false
	}
	if stamp, ok := indexStamp(string(entries[1])); !ok || stamp != string(entries[2]) {
		return trackedList{}, false
	}
	return trackedList{files: splitNUL(entries[3]), ok: true}, true
}

// splitNUL splits git's NUL-terminated `-z` output into its entries,
// dropping empty ones.
func splitNUL(out []byte) []string {
	var entries []string
	for _, entry := range bytes.Split(out, []byte{0}) {
		if len(entry) > 0 {
			entries = append(entries, string(entry))
		}
	}
	return entries
}
