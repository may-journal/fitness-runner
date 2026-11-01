// Package repocheck finds a repo's own checks (ADR 0009, ADR 0012) and builds
// the Go ones, as docs/local-checks.md describes.
//
//fitness:topic local-checks
package repocheck

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Folder is the folder at a repo's root that holds its own checks, read
// with no config.
const Folder = "fitness"

// Program is the file name of a folder check's executable entry point.
const Program = "check"

// Readme is the file every folder check documents its rule in.
const Readme = "README.md"

// flatPrefix starts the file name of a flat program in the localChecks
// folder.
const flatPrefix = "fitness-check-"

// checkName is the shape of a check name: lowercase words joined by dashes.
var checkName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// paired matches the name of a check paired with a doc: adr-NNNN keeps an
// architecture decision, and req-NNNN proves a requirement.
var paired = regexp.MustCompile(`^(adr|req)-\d{4}$`)

// Paired reports whether a check of this name keeps an ADR or proves a
// requirement, where judging no files means a broken scan.
func Paired(name string) bool {
	return paired.MatchString(name)
}

// skipped are the folder names a checks folder may hold beside its checks:
// Go's shared-code and fixture folders.
var skipped = []string{"internal", "testdata"}

// Check is one of a repo's own checks.
type Check struct {
	// Name is the check name, from the folder or program name.
	Name string
	// Path is the program to run. It is empty for a Go check until Build
	// writes one.
	Path string
	// Shown is where the check lives, relative to the repo root, as a
	// person reads it in a message.
	Shown string
	// Folder is a folder check's folder relative to the repo root, or empty
	// for a flat program.
	Folder string
	// Readme is the check's README.md, or empty for a flat program.
	Readme string
	// Source is the folder of Go source to build, or empty when Path is the
	// program.
	Source string
	// Module is the folder holding the go.mod Source builds with.
	Module string
}

// Find maps by name each check in root's fitness/ and configured localChecks
// folders, failing on an empty folder or duplicate name.
func Find(root, configured string) (map[string]Check, error) {
	checks := map[string]Check{}
	if err := addDefault(checks, root, configured); err != nil {
		return nil, err
	}
	if configured == "" {
		return checks, nil
	}
	dir, err := localDir(root, configured)
	if err != nil {
		return nil, err
	}
	return checks, addFolder(checks, root, dir, configured)
}

// addDefault adds the checks in root's fitness/ folder, when it has one
// that localChecks does not already name.
func addDefault(checks map[string]Check, root, configured string) error {
	dir := filepath.Join(root, Folder)
	if configured == Folder || !isFolder(dir) {
		return nil
	}
	if !insideRoot(root, dir) {
		return fmt.Errorf("%s/ links outside the repo; commit the checks folder itself", Folder)
	}
	return addFolder(checks, root, dir, Folder)
}

// addFolder adds every check in dir, shown as shown, failing on the first
// that cannot run, or when dir holds none.
func addFolder(checks map[string]Check, root, dir, shown string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("%s: %w", shown, err)
	}
	before := len(checks)
	for _, entry := range entries {
		if err := addEntry(checks, root, dir, shown, entry); err != nil {
			return err
		}
	}
	if len(checks) == before {
		return fmt.Errorf("%s holds no check; add %s/<name>/ with a %s and a %s program, or remove the folder", shown, shown, Readme, Program)
	}
	return nil
}

// addEntry adds the check one entry of a checks folder holds, if any,
// failing on a name another entry already took.
func addEntry(checks map[string]Check, root, dir, shown string, entry os.DirEntry) error {
	c, ok, err := entryCheck(root, dir, shown, entry)
	if err != nil || !ok {
		return err
	}
	if prior, taken := checks[c.Name]; taken {
		return fmt.Errorf("local check %s is both %s and %s; keep one", c.Name, prior.Shown, c.Shown)
	}
	checks[c.Name] = c
	return nil
}

// entryCheck reads one checks-folder entry as a folder check or a flat program,
// or neither for a superseded ADR folder.
func entryCheck(root, dir, shown string, entry os.DirEntry) (Check, bool, error) {
	path := filepath.Join(dir, entry.Name())
	if !entry.IsDir() {
		return flatCheck(root, path, shown, entry.Name())
	}
	if notCheck(path) {
		return Check{}, false, nil
	}
	c, err := folderCheck(root, dir, shown, entry.Name())
	return c, err == nil, err
}

// flatCheck reads a fitness-check-<name> program; any other file is left
// alone.
func flatCheck(root, path, shown, file string) (Check, bool, error) {
	name, ok := strings.CutPrefix(file, flatPrefix)
	if !ok {
		return Check{}, false, nil
	}
	c := Check{Name: name, Path: path, Shown: filepath.ToSlash(filepath.Join(shown, file))}
	if err := named(c, shown); err != nil {
		return Check{}, false, err
	}
	return c, true, runnable(root, c)
}

// folderCheck reads the folder check name in dir: its README and its
// program, or the Go source fitness builds into one.
func folderCheck(root, dir, shown, name string) (Check, error) {
	folder := filepath.Join(dir, name)
	c := Check{Name: name, Shown: filepath.ToSlash(filepath.Join(shown, name)), Readme: filepath.Join(folder, Readme)}
	c.Folder = c.Shown
	if err := named(c, shown); err != nil {
		return Check{}, err
	}
	if !isFile(c.Readme) {
		return Check{}, fmt.Errorf("local check %s has no %s; add one that states the rule and how to pass it", c.Shown, Readme)
	}
	if program := filepath.Join(folder, Program); exists(program) {
		c.Path = program
		c.Shown += "/" + Program
		return c, runnable(root, c)
	}
	return goCheck(root, dir, shown, c)
}

// goCheck reads a folder check with no program as Go source that fitness
// builds with the go.mod beside the folder.
func goCheck(root, dir, shown string, c Check) (Check, error) {
	folder := filepath.Dir(c.Readme)
	switch {
	case !hasGo(folder):
		return Check{}, fmt.Errorf("local check %s has no program; add an executable %s or a main.go", c.Shown, Program)
	case !isFile(filepath.Join(dir, "go.mod")):
		return Check{}, fmt.Errorf("local check %s is Go source, so %s/go.mod must exist; run go mod init in %s and require github.com/may-journal/fitness-runner/go", c.Shown, shown, shown)
	case !insideRoot(root, folder):
		return Check{}, fmt.Errorf("local check %s links outside the repo; commit the source itself", c.Shown)
	}
	c.Source, c.Module = folder, dir
	return c, nil
}

// named fails a check whose name is not lowercase words joined by dashes.
func named(c Check, shown string) error {
	if checkName.MatchString(c.Name) {
		return nil
	}
	return fmt.Errorf("local check %s: a check name is lowercase words joined by dashes, so rename it or move it out of %s", c.Shown, shown)
}

// runnable fails a program that is not an executable file inside the repo,
// naming the fix.
func runnable(root string, c Check) error {
	info, err := os.Stat(c.Path)
	if err != nil {
		return fmt.Errorf("local check %s: %w", c.Shown, err)
	}
	if info.IsDir() || info.Mode()&0o111 == 0 {
		return fmt.Errorf("local check %s is not an executable file; run chmod +x %s", c.Shown, c.Shown)
	}
	if !insideRoot(root, c.Path) {
		return fmt.Errorf("local check %s links outside the repo; commit the program itself", c.Shown)
	}
	return nil
}

// hasGo reports whether folder holds Go source other than tests.
func hasGo(folder string) bool {
	paths, _ := filepath.Glob(filepath.Join(folder, "*.go"))
	return slices.ContainsFunc(paths, func(p string) bool { return !strings.HasSuffix(p, "_test.go") })
}

// localDir resolves the configured folder against root and refuses one outside
// the repo or missing, since CI cannot run untracked checks.
func localDir(root, configured string) (string, error) {
	if !filepath.IsLocal(filepath.FromSlash(configured)) {
		return "", outsideRepo(configured)
	}
	dir := filepath.Join(root, filepath.FromSlash(configured))
	if err := isDir(dir, configured); err != nil {
		return "", err
	}
	if !insideRoot(root, dir) {
		return "", outsideRepo(configured)
	}
	return dir, nil
}

// isDir fails a configured localChecks path that is missing or not a
// directory.
func isDir(dir, configured string) error {
	info, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf(".fitnessrc.json localChecks directory %q does not exist", configured)
	}
	if err == nil && !info.IsDir() {
		err = fmt.Errorf(".fitnessrc.json localChecks %q is not a directory", configured)
	}
	return err
}

// outsideRepo is the error for a localChecks folder outside the repo.
func outsideRepo(configured string) error {
	return fmt.Errorf(".fitnessrc.json localChecks %q must be a path inside the repo, relative to its root", configured)
}

// insideRoot reports whether path, with every symlink followed, stays inside
// root, so a link cannot run a program the repo does not track.
func insideRoot(root, path string) bool {
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(realRoot, real)
	return err == nil && filepath.IsLocal(rel)
}

// isFolder reports whether path is a folder.
func isFolder(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// isFile reports whether path is a regular file.
func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// exists reports whether anything is at path.
func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
