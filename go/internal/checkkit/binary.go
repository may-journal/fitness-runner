package checkkit

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BinPrefix starts the name every built-in check runs under, and every
// local check's file name.
const BinPrefix = "fitness-check-"

// Program is how to start one check. Path is the file to exec. Name, when
// set, is the program name the check starts under: the one fitness
// executable runs the built-in check that name names.
type Program struct {
	Path, Name string
}

// Command returns the command that runs the program with args.
func (p Program) Command(args ...string) *exec.Cmd {
	cmd := exec.Command(p.Path, args...)
	if p.Name != "" {
		cmd.Args[0] = p.Name
	}
	return cmd
}

// Builtin returns the program for a built-in check: the running fitness
// executable, started as fitness-check-<name>. Every check runs from that
// one file, so a new install loads one binary instead of one per check.
func Builtin(name string) (Program, error) {
	self, err := os.Executable()
	if err != nil {
		return Program{}, fmt.Errorf("cannot find the fitness executable to run %s%s: %w", BinPrefix, name, err)
	}
	return Program{Path: self, Name: BinPrefix + name}, nil
}

// Self returns the program that starts the running check again: this
// executable, under the name it was started with, since one fitness
// executable holds every check and the name picks which one runs.
func Self() (Program, error) {
	self, err := os.Executable()
	if err != nil {
		return Program{}, err
	}
	return Program{Path: self, Name: filepath.Base(os.Args[0])}, nil
}

// Beside maps the name of each fitness-check-<name> program beside the
// running fitness executable to its path. An installed bundle links every
// built-in check there to fitness itself; those links are the built-in
// checks, so they are left out. Any other program there adds a check, or
// replaces the built-in check of its name. None is no error, as after
// `go install`.
func Beside() map[string]string {
	self, err := os.Executable()
	if err != nil {
		return nil
	}
	real, _ := filepath.EvalSymlinks(self)
	paths, _ := filepath.Glob(filepath.Join(filepath.Dir(self), BinPrefix+"*"))
	programs := map[string]string{}
	for _, p := range paths {
		if target, _ := filepath.EvalSymlinks(p); executableFile(p) && target != real {
			programs[strings.TrimPrefix(filepath.Base(p), BinPrefix)] = p
		}
	}
	return programs
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
