package distribution

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// checkLink marks a bundle file that is a link to the fitness binary, in
// place of a content hash. Links are recorded with the hashes, so a cached
// bundle missing one fails validation and is unpacked again.
const checkLink = "link:fitness"

// listTimeout bounds `fitness check --list`, which only prints names, so a
// bundle binary that hangs fails the install instead of stalling it.
const listTimeout = 30 * time.Second

// linkChecks links a fitness-check-<name> name to the fitness binary for every
// listed check, so callers using a program name keep working.
func linkChecks(directory string, hashes map[string]string) error {
	if shipsCheckBinaries(hashes) {
		return nil
	}
	names, err := listChecks(directory)
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := linkCheck(directory, name, hashes); err != nil {
			return err
		}
	}
	return nil
}

// shipsCheckBinaries reports whether a bundle holds a fitness-check-<name>
// binary of its own, as bundles from before the one binary do.
func shipsCheckBinaries(hashes map[string]string) bool {
	for name := range hashes {
		if strings.HasPrefix(name, checkkit.BinPrefix) {
			return true
		}
	}
	return false
}

// listChecks names the checks built into the bundle's fitness binary.
func listChecks(directory string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, filepath.Join(directory, "fitness"), "check", "--list").Output()
	if err != nil {
		return nil, fmt.Errorf("list the bundle's checks: %w", err)
	}
	return strings.Fields(string(out)), nil
}

// linkCheck links fitness-check-<name> to the fitness binary in directory
// and records the link with the bundle's hashes.
func linkCheck(directory, name string, hashes map[string]string) error {
	link := checkkit.BinPrefix + name
	if !binaryPattern.MatchString(link) {
		return fmt.Errorf("unsafe check name %q", name)
	}
	hashes[link] = checkLink
	return os.Symlink("fitness", filepath.Join(directory, link))
}

// validLink reports whether path is a link to the bundle's fitness binary.
func validLink(path string) bool {
	target, err := os.Readlink(path)
	return err == nil && target == "fitness"
}
