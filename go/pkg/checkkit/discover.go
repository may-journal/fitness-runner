package checkkit

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Protocol is the version of this contract the runner and each check speak.
// A check whose --describe declares another fails the run before any check
// starts.
const Protocol = 1

// ProtocolEnv names the variable through which the runner tells each check
// the protocol version it speaks.
const ProtocolEnv = "FITNESS_PROTOCOL"

// ChecksDirEnv names the folder of check programs that Discover searches
// first, ahead of the folder beside fitness and PATH.
const ChecksDirEnv = "FITNESS_CHECKS_DIR"

// ProtocolOf is the protocol version d declares, 1 when it declares none.
func ProtocolOf(d Describe) int {
	if d.Protocol == 0 {
		return 1
	}
	return d.Protocol
}

// Discover maps each installed fitness-check-<name> program to its path: in
// the ChecksDirEnv folder, beside fitness, then in each absolute PATH folder.
// The first of a name wins, and a link to a fitness executable never counts.
func Discover() map[string]string {
	self, _ := os.Executable()
	real, _ := filepath.EvalSymlinks(self)
	programs := map[string]string{}
	for _, folder := range searchFolders(self) {
		for name, path := range programsIn(folder, real) {
			if _, found := programs[name]; !found {
				programs[name] = path
			}
		}
	}
	return programs
}

// searchFolders lists the absolute folders Discover searches, in order. A
// relative PATH folder, such as ".", would run a program the repo holds.
func searchFolders(self string) []string {
	var folders []string
	if configured := os.Getenv(ChecksDirEnv); configured != "" {
		// A configured folder counts from where fitness starts.
		abs, _ := filepath.Abs(configured)
		folders = append(folders, abs)
	}
	if self != "" {
		folders = append(folders, filepath.Dir(self))
	}
	return slices.DeleteFunc(append(folders, filepath.SplitList(os.Getenv("PATH"))...), func(folder string) bool {
		return !filepath.IsAbs(folder)
	})
}

// programsIn maps each executable fitness-check-<name> file in folder to its
// path. It skips a link to fitness, the running real one or any other.
func programsIn(folder, real string) map[string]string {
	paths, _ := filepath.Glob(filepath.Join(folder, BinPrefix+"*"))
	programs := map[string]string{}
	for _, p := range paths {
		target, _ := filepath.EvalSymlinks(p)
		if executableFile(p) && target != real && !isFitness(target) {
			programs[strings.TrimPrefix(filepath.Base(p), BinPrefix)] = p
		}
	}
	return programs
}

// isFitness reports whether path names a fitness executable, which runs a
// built-in check under a link's name.
func isFitness(path string) bool {
	base := strings.TrimSuffix(filepath.Base(path), ".exe")
	return base == "fitness"
}
