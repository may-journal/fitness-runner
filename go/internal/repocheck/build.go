package repocheck

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Build returns the program for a Go check: the one in cache built from the
// same source, or a new one, built with `go build` in the check's module.
// The cache key holds every Go source, go.mod and go.sum file in the
// module, Go's version and platform, and version, the running fitness's, so
// a warm run costs one hash and no build. Cold, it says on log that it
// builds. A missing go names how to get one.
func Build(c Check, cache, version string, log io.Writer) (string, error) {
	if c.Source == "" {
		return c.Path, nil
	}
	if _, err := exec.LookPath("go"); err != nil {
		return "", fmt.Errorf("local check %s is Go source, and go is not on PATH; install Go from https://go.dev/dl, or commit an executable %s/%s instead", c.Shown, c.Shown, Program)
	}
	key, err := buildKey(c, version)
	if err != nil {
		return "", fmt.Errorf("local check %s: %w", c.Shown, err)
	}
	out := filepath.Join(cache, "checks", key, c.Name)
	if isFile(out) {
		return out, nil
	}
	fmt.Fprintf(log, "fitness: building %s with go\n", c.Shown)
	return out, goBuild(c, out)
}

// goBuild builds the check's package into out, through a temporary file
// renamed into place, so a run beside it never starts a half-written
// program.
func goBuild(c Check, out string) error {
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(out), c.Name+".*.tmp")
	if err != nil {
		return err
	}
	_ = tmp.Close()
	defer os.Remove(tmp.Name())
	cmd := exec.Command("go", "build", "-o", tmp.Name(), "./"+filepath.Base(c.Source))
	cmd.Dir = c.Module
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("local check %s does not build; fix it so `go -C %s build ./%s` passes:\n%s", c.Shown, filepath.Dir(c.Shown), filepath.Base(c.Shown), strings.TrimSpace(output.String()))
	}
	return os.Rename(tmp.Name(), out)
}

// buildKey hashes what a built program depends on into a short name.
func buildKey(c Check, version string) (string, error) {
	h := sha256.New()
	env, err := exec.Command("go", "env", "GOVERSION", "GOOS", "GOARCH", "GOFLAGS").Output()
	if err != nil {
		return "", fmt.Errorf("go env: %w", err)
	}
	fmt.Fprintf(h, "%s\x00%s\x00%s\x00", version, env, filepath.Base(c.Source))
	err = filepath.WalkDir(c.Module, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return hiddenDir(c.Module, path, d.Name())
		}
		return hashInput(h, c.Module, path)
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

// hiddenDir skips a dot folder below the module, which go never builds.
func hiddenDir(module, path, name string) error {
	if path != module && strings.HasPrefix(name, ".") {
		return filepath.SkipDir
	}
	return nil
}

// hashInput hashes path into h when it can change what go builds.
func hashInput(h io.Writer, module, path string) error {
	if !buildInput(filepath.Base(path)) {
		return nil
	}
	return hashFile(h, module, path)
}

// buildInput reports whether a file of name can change what go builds.
func buildInput(name string) bool {
	return strings.HasSuffix(name, ".go") || name == "go.mod" || name == "go.sum"
}

// hashFile writes path, relative to module, and its content into h.
func hashFile(h io.Writer, module, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(module, path)
	if err != nil {
		return err
	}
	fmt.Fprintf(h, "%s\x00%d\x00", filepath.ToSlash(rel), len(data))
	_, err = h.Write(data)
	return err
}
