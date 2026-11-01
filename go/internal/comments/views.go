package comments

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/may-journal/fitness-runner/go/internal/par"
)

// ViewsEnv names the folder where the runner left each file's code view and
// comment view, built once for every check of a run.
const ViewsEnv = "FITNESS_VIEWS_DIR"

// Files inside the views folder: every view back to back, and the index that
// finds each one.
const (
	packFile  = "views"
	indexFile = "index.json"
)

// Shared is one run's views folder: Env hands it to checks, and Remove
// deletes it.
type Shared struct {
	Env    []string
	Remove func()
}

// viewIndex is what the runner read: its root and each file's views.
type viewIndex struct {
	Root  string           `json:"root"`
	Files map[string]entry `json:"files"`
}

// entry places one file's views in the pack, with the time and size the
// file had when it was read.
type entry struct {
	Mod      int64  `json:"mod"`
	Size     int64  `json:"size"`
	Code     packed `json:"code"`
	Comments packed `json:"comments"`
}

// packed is the offset and length of one view in the pack. A zero length
// marks no comment view, as Markdown has.
type packed struct {
	At  int64 `json:"at"`
	Len int64 `json:"len"`
}

// built is one file's views, read once.
type built struct {
	rel            string
	info           fs.FileInfo
	code, comments []byte
}

// ShareViews reads each of files under root once and writes both its views
// to a temporary folder for the run's checks. On failure, checks build their
// own views.
func ShareViews(root string, files []string) Shared {
	none := Shared{Remove: func() {}}
	views := par.Map(len(files), 0, func(i int) built { return buildViews(root, files[i]) })
	dir, err := os.MkdirTemp("", "fitness-views-*")
	if err != nil {
		return none
	}
	remove := func() { _ = os.RemoveAll(dir) }
	if writePack(dir, filepath.Clean(root), views) != nil {
		remove()
		return none
	}
	return Shared{Env: []string{ViewsEnv + "=" + dir}, Remove: remove}
}

// buildViews reads the file at rel and builds both its views. A file no
// reader covers gets none, as it is its own code view.
func buildViews(root, rel string) built {
	info, data, ok := readViewed(root, rel)
	if !ok || path.Ext(rel) != ".md" && Lang(rel, data) == "" {
		return built{}
	}
	b := built{rel: rel, info: info, code: CodeView(rel, data)}
	if path.Ext(rel) != ".md" {
		blocks := Source(rel, data)
		placeSource(blocks, data)
		b.comments, _ = json.Marshal(blocks)
	}
	return b
}

// readViewed reads the regular file at rel when its views differ from its
// text: Markdown, or a file a comment reader covers.
func readViewed(root, rel string) (fs.FileInfo, []byte, bool) {
	if path.Ext(rel) != ".md" && !covered(rel) {
		return nil, nil, false
	}
	src := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(src)
	if err != nil || !info.Mode().IsRegular() {
		return nil, nil, false
	}
	data, err := os.ReadFile(src)
	return info, data, err == nil
}

// writePack writes the views back to back in one file, and the index.
func writePack(dir, root string, views []built) error {
	var pack bytes.Buffer
	index := viewIndex{Root: root, Files: map[string]entry{}}
	for _, v := range views {
		if v.info == nil {
			continue
		}
		e := entry{Mod: v.info.ModTime().UnixNano(), Size: v.info.Size()}
		e.Code = packed{int64(pack.Len()), int64(len(v.code))}
		pack.Write(v.code)
		e.Comments = packed{int64(pack.Len()), int64(len(v.comments))}
		pack.Write(v.comments)
		index.Files[v.rel] = e
	}
	encoded, err := json.Marshal(index)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, packFile), pack.Bytes(), 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, indexFile), encoded, 0o600)
}

// CodeFile is the code view of the file at rel under root, from the runner's
// view while it matches the file. An empty root reads rel as a full path.
func CodeFile(root, rel string) ([]byte, error) {
	src := filepath.Join(root, filepath.FromSlash(rel))
	if e, ok := sharedEntry(root, rel, src); ok {
		if data, ok := readSpan(e.Code); ok {
			return data, nil
		}
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return nil, err
	}
	return CodeView(rel, data), nil
}

// sharedBlocks returns the placed comment view of the file at rel from the
// runner's views, or false when none matches the file.
func sharedBlocks(root, rel string) ([]Block, bool) {
	e, ok := sharedEntry(root, rel, filepath.Join(root, filepath.FromSlash(rel)))
	if !ok {
		return nil, false
	}
	data, ok := readSpan(e.Comments)
	var blocks []Block
	if !ok || json.Unmarshal(data, &blocks) != nil {
		return nil, false
	}
	for i := range blocks {
		blocks[i].placed = true
	}
	return blocks, true
}

// sharedEntry is the runner's entry for rel under root, while the file at
// src keeps the time and size it had.
func sharedEntry(root, rel, src string) (entry, bool) {
	shared := loadShared()
	e, ok := shared.index.Files[rel]
	if !ok || shared.index.Root != filepath.Clean(root) {
		return entry{}, false
	}
	info, err := os.Stat(src)
	return e, err == nil && info.ModTime().UnixNano() == e.Mod && info.Size() == e.Size
}

// readSpan reads one view from the pack, or false for an empty one.
func readSpan(s packed) ([]byte, bool) {
	if s.Len == 0 {
		return nil, false
	}
	data := make([]byte, s.Len)
	_, err := loadShared().pack.ReadAt(data, s.At)
	return data, err == nil
}

// sharedViews is the runner's views as this process opened them.
type sharedViews struct {
	index viewIndex
	pack  *os.File
}

// loadShared opens the runner's views once per process; with none, its
// index holds no files.
var loadShared = sync.OnceValue(func() sharedViews {
	dir := os.Getenv(ViewsEnv)
	if dir == "" {
		return sharedViews{}
	}
	var shared sharedViews
	data, err := os.ReadFile(filepath.Join(dir, indexFile))
	if err != nil || json.Unmarshal(data, &shared.index) != nil {
		return sharedViews{}
	}
	if shared.pack, err = os.Open(filepath.Join(dir, packFile)); err != nil {
		return sharedViews{}
	}
	return shared
})
