package walkfs

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
	"github.com/may-journal/fitness-runner/go/internal/par"
)

// TrackedFilesEnv names the variable handing every check the file ShareTracked
// wrote, so a run lists tracked files once.
const TrackedFilesEnv = "FITNESS_TRACKED_FILES"

// trackedMemo holds each root's tracked-file list for the process, so a walk
// never waits on git for another root and asks git once.
var trackedMemo = struct {
	sync.Mutex
	roots map[string]*trackedEntry
}{roots: map[string]*trackedEntry{}}

// trackedEntry is one root's list, filled once.
type trackedEntry struct {
	once sync.Once
	list trackedList
}

// trackedList is one root's tracked paths; ok is false outside git, and onDisk
// marks a list needing no stat.
type trackedList struct {
	files  []string
	ok     bool
	onDisk bool
}

// trackedFiles returns root's slash-separated tracked paths, preferring the
// runner's shared list; ok is false outside a git work tree.
func trackedFiles(root string) trackedList {
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
	return entry.list
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

// Shared is one run's tracked-file list. Env hands it to checks, Files counts
// its paths, Paths lists them for the runner, and Remove deletes it.
type Shared struct {
	Env    []string
	Files  int
	Paths  []string
	Remove func()
}

// ShareTracked writes root's tracked regular files to a temporary file so no
// check stats them again; on failure checks list their own.
func ShareTracked(root string) (Shared, error) {
	none := Shared{Remove: func() {}}
	content, paths, err := trackedSnapshot(filepath.Clean(root))
	if content == nil {
		return none, err
	}
	f, err := os.CreateTemp("", "fitness-tracked-*")
	if err != nil {
		return none, nil
	}
	remove := func() { _ = os.Remove(f.Name()) }
	_, writeErr := f.Write(content)
	if closeErr := f.Close(); writeErr != nil || closeErr != nil {
		remove()
		return none, nil
	}
	return Shared{Env: []string{TrackedFilesEnv + "=" + f.Name()}, Files: len(paths), Paths: paths, Remove: remove}, nil
}

// trackedSnapshot is the shared file's content for root and its paths, or
// nil when git or the index cannot be read.
func trackedSnapshot(root string) ([]byte, []string, error) {
	index, err := gitx.Repo(root).GitPath("index")
	if err != nil {
		return nil, nil, stalledOnly(err)
	}
	// The stamp is read before the listing, so an index rewritten in
	// between only makes checks list their own files.
	stamp, ok := indexStamp(index)
	entries, err := gitx.Repo(root).IndexEntries()
	if err != nil || !ok {
		return nil, nil, stalledOnly(err)
	}
	files := onDisk(root, regularEntries(entries))
	return encodeTracked([]string{root, index, stamp}, files), files, nil
}

// regularEntries keeps the paths whose index mode is a regular file, once
// each, dropping symlinks and submodules by mode alone. An unmerged path's
// later stages follow its first, so only the first is kept.
func regularEntries(entries []gitx.IndexEntry) []string {
	paths := make([]string, 0, len(entries))
	for i, e := range entries {
		if regularMode(e.Mode) && (i == 0 || entries[i-1].Path != e.Path) {
			paths = append(paths, e.Path)
		}
	}
	return paths
}

// regularMode reports whether an index mode names a regular file.
func regularMode(mode string) bool {
	return mode == "100644" || mode == "100755"
}

// onDisk keeps the paths that are still regular files under root, stating
// them in parallel workers, one per CPU, in their input order.
func onDisk(root string, paths []string) []string {
	present := par.Map(len(paths), 0, func(i int) bool { return RegularFile(root, paths[i]) })
	kept := make([]string, 0, len(paths))
	for i, path := range paths {
		if present[i] {
			kept = append(kept, path)
		}
	}
	return kept
}

// stalledOnly keeps err only when git stalled at its deadline, since any other
// failure just means checks list their own files.
func stalledOnly(err error) error {
	if gitx.Stalled(err) {
		return err
	}
	return nil
}

// indexStamp names the index's version by inode, mtime and size, because git
// renames a new index into place.
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

// sharedTracked reads the runner's list when it names root and the index is
// unchanged, ignoring any other.
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
	return trackedList{files: splitNUL(entries[3]), ok: true, onDisk: true}, true
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
