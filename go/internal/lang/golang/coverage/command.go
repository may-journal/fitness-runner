package coverage

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang/toolchain"
)

type runner struct {
	ctx            context.Context
	dir, bin, temp string
	label          string
}

func (r runner) command(args ...string) ([]byte, error) {
	return r.run("go "+strings.Join(args, " "), nil, args...)
}

// run executes go with args in the module directory and returns its combined
// output, as exec does.
func (r runner) run(label string, env []string, args ...string) ([]byte, error) {
	return r.exec(r.bin, r.dir, label, env, args...)
}

// exec runs bin with args in dir and returns its combined output. A failure is
// labeled; it is a checkkit.Internal error when the environment stopped the
// run (the binary could not start, a signal other than the deadline ended it,
// or sandbox text), while the run's deadline stays a plain error.
func (r runner) exec(bin, dir, label string, env []string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(r.ctx, bin, args...)
	cmd.Dir = dir
	cmd.WaitDelay = 2 * time.Second
	cmd.Env = append(append(toolchain.CleanEnv(os.Environ()), "GOWORK=off"), env...)
	o := checkkit.ExecCombined(cmd)
	if o.Err == nil {
		return []byte(o.Stdout), nil
	}
	err := fmt.Errorf("%s: %w\n%s", label, o.Err, o.Stdout)
	if r.ctx.Err() != nil {
		return nil, err
	}
	if o.EnvErr(label) != nil || toolchain.SandboxErr(label, o.Code, o.Stdout) != nil {
		return nil, checkkit.Internal(err)
	}
	return nil, err
}

// measure runs go test over pkg once with a fresh profile.
func (r runner) measure(pkg, pattern string) (Profile, error) {
	return r.collect(func(profile, child string) error {
		args := []string{"test", "-count=1", "-covermode=atomic", "-coverpkg=./...", "-coverprofile=" + profile, "-run=" + pattern, pkg}
		_, err := r.run("go test coverage", []string{"FITNESS_GO_COVER_DIR=" + child}, args...)
		return err
	})
}

// collect gives one measurement a private profile path and child coverage
// directory, runs it, and returns the merged profile.
func (r runner) collect(measure func(profile, child string) error) (Profile, error) {
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
	if err := measure(profile, child); err != nil {
		return nil, err
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
	if err := matchingBlocks(p, other); err != nil {
		return err
	}
	return p.Merge(other)
}

func matchingBlocks(expected, child Profile) error {
	for b := range child {
		if _, ok := expected[b]; !ok {
			return fmt.Errorf("child coverage differs from test instrumentation in %s", b.File)
		}
	}
	return nil
}
