package distribution

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// checkLink marks a bundle file that is a link to the fitness binary, in
// place of a content hash. Links are recorded with the hashes, so a cached
// bundle missing one fails validation and is unpacked again.
const checkLink = "link:fitness"

// linkChecks gives a one-binary bundle a fitness-check-<name> link to its
// fitness binary for every check `fitness check --list` names, so a caller
// that runs a check by its own program name keeps working. An older bundle
// that ships one binary per check gets no links.
func linkChecks(directory string, hashes map[string]string) error {
	if len(hashes) != 1 {
		return nil
	}
	out, err := exec.Command(filepath.Join(directory, "fitness"), "check", "--list").Output()
	if err != nil {
		return fmt.Errorf("list the bundle's checks: %w", err)
	}
	for _, name := range strings.Fields(string(out)) {
		if err := linkCheck(directory, name, hashes); err != nil {
			return err
		}
	}
	return nil
}

// linkCheck links fitness-check-<name> to the fitness binary in directory
// and records the link with the bundle's hashes.
func linkCheck(directory, name string, hashes map[string]string) error {
	link := "fitness-check-" + name
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
