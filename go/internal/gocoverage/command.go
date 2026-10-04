package gocoverage

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/toolchain"
)

type runner struct {
	ctx            context.Context
	dir, bin, temp string
}

func (r runner) command(args ...string) ([]byte, error) {
	cmd := exec.CommandContext(r.ctx, r.bin, args...)
	cmd.Dir = r.dir
	cmd.Env = append(toolchain.CleanEnv(os.Environ()), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return out, nil
}
func (r runner) measure(pkg, pattern string) (Profile, error) {
	dir, err := os.MkdirTemp(r.temp, "run-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	profile := filepath.Join(dir, "coverage.out")
	child := filepath.Join(dir, "children")
	if err := os.Mkdir(child, 0700); err != nil {
		return nil, err
	}
	args := []string{"test", "-count=1", "-covermode=atomic", "-coverpkg=./...", "-coverprofile=" + profile, "-run=" + pattern, pkg}
	cmd := exec.CommandContext(r.ctx, r.bin, args...)
	cmd.Dir = r.dir
	cmd.Env = append(toolchain.CleanEnv(os.Environ()), "GOWORK=off", "FITNESS_GO_COVER_DIR="+child)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go test coverage: %w\n%s", err, out)
	}
	return r.profiles(profile, child)
}
func (r runner) profiles(file, child string) (Profile, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	p, err := ReadProfile(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return p, r.mergeChildren(p, child)
}
func (r runner) mergeChildren(p Profile, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return err
	}
	file := filepath.Join(filepath.Dir(dir), "children.out")
	if _, err := r.command("tool", "covdata", "textfmt", "-i="+dir, "-o="+file); err != nil {
		return err
	}
	return mergeProfileFile(p, file)
}
func mergeProfileFile(p Profile, file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	other, err := ReadProfile(bytes.NewReader(data))
	if err != nil {
		return err
	}
	return p.Merge(other)
}
