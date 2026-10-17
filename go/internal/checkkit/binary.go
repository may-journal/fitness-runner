package checkkit

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// binPrefix starts every check binary's file name.
const binPrefix = "fitness-check-"

// Binary locates a bundled check beside the running Fitness executable.
func Binary(name string) (string, error) {
	binName := binPrefix + name
	if self, err := os.Executable(); err == nil {
		path := filepath.Join(filepath.Dir(self), binName)
		if executableFile(path) {
			return path, nil
		}
	}
	return "", fmt.Errorf("%s is not installed beside fitness", binName)
}

// Installed names every check installed beside the running Fitness
// executable, without the binary prefix, sorted.
func Installed() []string {
	self, err := os.Executable()
	if err != nil {
		return nil
	}
	paths, _ := filepath.Glob(filepath.Join(filepath.Dir(self), binPrefix+"*"))
	var names []string
	for _, p := range paths {
		if executableFile(p) {
			names = append(names, strings.TrimPrefix(filepath.Base(p), binPrefix))
		}
	}
	return names
}

// executableFile reports whether path is a file with an execute bit set.
func executableFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

// LastJSONLine returns the last nonempty stdout line after any tool noise.
func LastJSONLine(out []byte) []byte {
	lines := bytes.Split(bytes.TrimSpace(out), []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		if line := bytes.TrimSpace(lines[i]); len(line) > 0 {
			return line
		}
	}
	return nil
}
