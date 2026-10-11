package checkkit

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// Binary locates a bundled check beside the running Fitness executable.
func Binary(name string) (string, error) {
	binName := "fitness-check-" + name
	if self, err := os.Executable(); err == nil {
		path := filepath.Join(filepath.Dir(self), binName)
		if info, err := os.Stat(path); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return path, nil
		}
	}
	return "", fmt.Errorf("%s is not installed beside fitness", binName)
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
