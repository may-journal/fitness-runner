// Package sharedconf embeds the shared tool configs and materializes them to
// a content-keyed cache, the fallback after a repo's own and npm copies.
package sharedconf

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sync"
)

//go:embed config/*
var configFS embed.FS

// configDir is the embedded directory holding the shared config files.
const configDir = "config"

// installedRel is the shared npm package's config directory relative to a
// consumer repo ancestor — the compat path existing installs resolve through.
var installedRel = filepath.Join("node_modules", "@mayjournal", "fitness-shared", configDir)

// userCacheDir is os.UserCacheDir, a variable so tests can point the cache at
// a scratch directory.
var userCacheDir = os.UserCacheDir

// cacheKey is the content hash naming the cache directory, so changed files
// land in a fresh one an older binary is not reading.
var cacheKey = sync.OnceValue(func() string {
	h := sha256.New()
	for _, name := range fileNames() {
		data, _ := fs.ReadFile(configFS, path.Join(configDir, name))
		h.Write([]byte(name))
		h.Write([]byte{0})
		h.Write(data)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
})

// fileNames lists the embedded config filenames, sorted (the fs.ReadDir
// contract), so the content hash is deterministic.
func fileNames() []string {
	entries, err := fs.ReadDir(configFS, configDir)
	if err != nil {
		return nil // unreachable: the directory is compiled into the binary
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	return names
}

// Materialize writes the embedded config directory to a content-keyed cache
// directory and returns its path, taking no lock since writers agree on bytes.
func Materialize() (string, error) {
	dir := filepath.Join(cacheBase(), "sharedconf-"+cacheKey())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("sharedconf: %w", err)
	}
	for _, name := range fileNames() {
		if err := materializeFile(dir, name); err != nil {
			return "", fmt.Errorf("sharedconf: %w", err)
		}
	}
	return dir, nil
}

// materializeFile writes the embedded config file name into dir unless present,
// since the content-keyed directory guarantees existing bytes are right.
func materializeFile(dir, name string) error {
	dst := filepath.Join(dir, name)
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	data, err := fs.ReadFile(configFS, path.Join(configDir, name))
	if err != nil {
		return err
	}
	return writeViaRename(dir, dst, data)
}

// cacheBase is the parent of every materialized config directory, falling back
// to the system temp dir when no user cache dir resolves.
func cacheBase() string {
	base, err := userCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "fitness")
}

// writeViaRename writes data to a temp file in dir, sets 0644, and renames it
// onto dst. Losing the rename race to a concurrent materialization is
// success: the winner wrote identical bytes.
func writeViaRename(dir, dst string, data []byte) error {
	name, err := writeTempFile(dir, dst, data)
	if err != nil {
		return err
	}
	if err := os.Rename(name, dst); err != nil {
		os.Remove(name)
		if _, statErr := os.Stat(dst); statErr == nil {
			return nil // a concurrent materialization won the rename
		}
		return err
	}
	return nil
}

// writeTempFile creates a temp file in dir named after dst, fills it with
// data, and returns its path, removing it on failure.
func writeTempFile(dir, dst string, data []byte) (string, error) {
	tmp, err := os.CreateTemp(dir, filepath.Base(dst)+".tmp*")
	if err != nil {
		return "", err
	}
	name := tmp.Name()
	if err := fillAndClose(tmp, data); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

// fillAndClose writes data to tmp, sets mode 0644, and closes it even on
// failure, leaving removal to the caller.
func fillAndClose(tmp *os.File, data []byte) error {
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	return tmp.Close()
}

// Resolve returns the path of config file filename: the repo's copy, else the
// nearest installed one, else the materialized embedded copy.
func Resolve(root, filename string) string {
	if p := filepath.Join(root, filename); regularFileExists(p) {
		return p
	}
	if p := installedPath(root, filename); p != "" {
		return p
	}
	dir, err := Materialize()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, filename)
}

// ResolveDir returns the directory holding the full shared config set: the
// nearest installed node_modules/@mayjournal/fitness-shared/config walking up
// from root, else the materialized embedded copy. There is no <root> step —
// callers that honor a repo-local config check the root themselves before
// asking for the shared directory. Empty only when materialization fails.
func ResolveDir(root string) string {
	if p := installedPath(root, ""); p != "" {
		return p
	}
	dir, err := Materialize()
	if err != nil {
		return ""
	}
	return dir
}

// installedPath walks up from root probing each ancestor's installed shared
// config for filename, or the directory when empty, returning "" if none has it.
func installedPath(root, filename string) string {
	dir := root
	if abs, err := filepath.Abs(root); err == nil {
		dir = abs
	}
	for {
		if p := filepath.Join(dir, installedRel, filename); pathExists(p) {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func regularFileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
