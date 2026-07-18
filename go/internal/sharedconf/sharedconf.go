// Package sharedconf ships the shared tool configuration directory inside
// the check binaries — the npm-free successor to the config/ directory the
// @mayjournal/fitness-shared npm package used to publish. The six files
// (constants.cjs, cspell.json, eslint.base.mjs, eslint.config.mjs,
// prettier.config.cjs, vitest.config.mjs) are embedded as frozen data and
// materialized on demand to a content-keyed cache directory — the whole
// directory at once, so the configs' relative imports ('./eslint.base.mjs',
// './constants.cjs', './cspell.json') keep working from the cache.
//
// Resolution contract (Resolve, ResolveDir): a consumer repo's own file wins;
// an installed node_modules/@mayjournal/fitness-shared/config, walking up
// from the root, comes second — existing npm installs keep winning; the
// embedded copy, materialized, is the fallback that makes every check work in
// a repo with no npm anywhere.
//
// Concurrency: Materialize is lock-free. The cache directory name is keyed by
// a hash of the embedded content, so a directory that exists under that name
// already holds exactly this content and racing writers all produce identical
// bytes. Each file is written to a temp file inside the cache directory and
// renamed into place; rename is atomic, so a reader never observes a partial
// file, and the per-file exists-check merely skips work another process has
// already finished.
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

// cacheKey is the short content hash naming the cache directory: any change
// to the embedded files lands in a fresh directory, never mutating one an
// older binary may be reading.
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

// Materialize writes the embedded config directory to
// <user cache>/fitness/sharedconf-<content hash>/ (os.TempDir when no user
// cache dir resolves) and returns the directory path. Files are written 0644;
// files already present are left alone — the content-keyed directory name
// guarantees they hold the right bytes — which makes repeat calls cheap and
// concurrent calls safe (see the package comment).
func Materialize() (string, error) {
	dir := filepath.Join(cacheBase(), "sharedconf-"+cacheKey())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("sharedconf: %w", err)
	}
	for _, name := range fileNames() {
		dst := filepath.Join(dir, name)
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		data, err := fs.ReadFile(configFS, path.Join(configDir, name))
		if err != nil {
			return "", fmt.Errorf("sharedconf: %w", err)
		}
		if err := writeViaRename(dir, dst, data); err != nil {
			return "", fmt.Errorf("sharedconf: %w", err)
		}
	}
	return dir, nil
}

// cacheBase is the parent of every materialized config directory: the user
// cache dir, or the system temp dir when none resolves (HOME unset, say).
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
	tmp, err := os.CreateTemp(dir, filepath.Base(dst)+".tmp*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
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

// Resolve returns the path a consumer check should hand its tool for the
// shared config file named filename: <root>/<filename> when the repo carries
// its own copy, else the nearest installed copy under
// node_modules/@mayjournal/fitness-shared/config walking up from root, else
// the embedded copy materialized to the cache. Empty only when that fallback
// materialization fails — callers degrade to their tool's own defaults.
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

// installedPath walks up from root (made absolute first) probing each
// ancestor's installed shared-config directory for filename — or for the
// directory itself when filename is empty. Empty when no install has it.
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
