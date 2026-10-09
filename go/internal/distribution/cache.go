package distribution

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var generationPattern = regexp.MustCompile(`^bundle-[0-9]+$`)

func writeAtomic(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".download-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func cachedArchive(path, digest string) ([]byte, bool) {
	data, err := readCacheFile(path, maxArchiveBytes)
	return data, err == nil && contentHash(data) == digest
}

func currentBundle(key string, hashes map[string]string) string {
	target, err := os.Readlink(filepath.Join(key, "current"))
	if err != nil || !generationPattern.MatchString(target) {
		return ""
	}
	path := filepath.Join(key, target)
	if validBundle(path, hashes) {
		return path
	}
	return ""
}

func validBundle(path string, hashes map[string]string) bool {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) != len(hashes) {
		return false
	}
	return validFiles(path, hashes)
}

func validFiles(path string, hashes map[string]string) bool {
	for name, hash := range hashes {
		if !validBinary(filepath.Join(path, name), hash) {
			return false
		}
	}
	return true
}

func validBinary(path, hash string) bool {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		return false
	}
	data, err := readCacheFile(path, maxBinaryBytes)
	return err == nil && contentHash(data) == hash
}

// Publish an immutable generation through an atomic pointer swap. Older
// generations stay available to processes already running their checks.
func publishBundle(key, staged string) (string, error) {
	name := filepath.Base(staged)
	pointer := filepath.Join(key, ".current-"+name)
	if err := os.Symlink(name, pointer); err != nil {
		return "", err
	}
	defer os.Remove(pointer)
	if err := os.Rename(pointer, filepath.Join(key, "current")); err != nil {
		return "", fmt.Errorf("publish cached bundle: %w", err)
	}
	return staged, nil
}

func readCacheFile(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return readBounded(file, limit)
}

// recordHashes saves a bundle generation's file hashes, taken from the
// verified archive, beside it, so later runs can check the installed files
// without reading or unpacking the archive again.
func recordHashes(key, generation string, hashes map[string]string) error {
	data, err := json.Marshal(hashes)
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(key, generation+".hashes.json"), data)
}

// recordedBundle returns the current bundle when every installed file still
// matches the hashes recorded at publish time, or "" so the caller verifies
// and unpacks the archive. A damaged file fails the match.
func recordedBundle(key string) string {
	target, err := os.Readlink(filepath.Join(key, "current"))
	if err != nil || !generationPattern.MatchString(target) {
		return ""
	}
	path := filepath.Join(key, target)
	if hashes := recorded(key, target); hashes == nil || !validBundle(path, hashes) {
		return ""
	}
	return path
}

// recorded reads the hashes saved for a generation, or nil when none are.
func recorded(key, generation string) map[string]string {
	var hashes map[string]string
	data, err := os.ReadFile(filepath.Join(key, generation+".hashes.json"))
	if err != nil || json.Unmarshal(data, &hashes) != nil {
		return nil
	}
	return hashes
}
