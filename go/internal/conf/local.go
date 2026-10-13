package conf

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// checkPrefix starts every check program's file name.
const checkPrefix = "fitness-check-"

// checkName is the shape of a check name: lowercase words joined by dashes.
var checkName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// LocalCheckBinaries maps each check in the repo's localChecks directory to
// its program, by check name. A config without localChecks returns nil. The
// directory must sit inside root and hold at least one check; every
// fitness-check-<name> entry in it must be an executable file. Other files
// in the directory are left alone.
func (c *Config) LocalCheckBinaries(root string) (map[string]string, error) {
	if c == nil || c.LocalChecks == "" {
		return nil, nil
	}
	dir, err := localDir(root, c.LocalChecks)
	if err != nil {
		return nil, err
	}
	return localPrograms(dir, c.LocalChecks)
}

// localPrograms maps each fitness-check-<name> entry in dir to its path,
// failing on the first one that cannot run, or when there is none.
func localPrograms(dir, configured string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("%s localChecks %q: %w", FileName, configured, err)
	}
	bins := map[string]string{}
	for _, entry := range entries {
		if err := addProgram(bins, dir, entry.Name(), configured); err != nil {
			return nil, err
		}
	}
	if len(bins) == 0 {
		return nil, fmt.Errorf("%s localChecks %q holds no %s<name> program; add one or remove localChecks", FileName, configured, checkPrefix)
	}
	return bins, nil
}

// addProgram adds the entry to bins when its file name marks a check.
func addProgram(bins map[string]string, dir, entry, configured string) error {
	name, ok := strings.CutPrefix(entry, checkPrefix)
	if !ok {
		return nil
	}
	path := filepath.Join(dir, entry)
	bins[name] = path
	return localProgram(path, name, configured)
}

// localDir resolves the configured directory against root and refuses one
// outside the repo or missing, since a check the repo does not track cannot
// run the same way in CI.
func localDir(root, configured string) (string, error) {
	if !filepath.IsLocal(filepath.FromSlash(configured)) {
		return "", fmt.Errorf("%s localChecks %q must be a path inside the repo, relative to its root", FileName, configured)
	}
	dir := filepath.Join(root, filepath.FromSlash(configured))
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("%s localChecks directory %q does not exist", FileName, configured)
	}
	if err == nil && !info.IsDir() {
		err = fmt.Errorf("%s localChecks %q is not a directory", FileName, configured)
	}
	return dir, err
}

// localProgram fails a local check whose name or file cannot run, naming
// the fix.
func localProgram(path, name, configured string) error {
	shown := filepath.ToSlash(filepath.Join(configured, checkPrefix+name))
	if !checkName.MatchString(name) {
		return fmt.Errorf("local check %s: a check name is lowercase words joined by dashes", shown)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("local check %s: %w", shown, err)
	}
	if info.IsDir() || info.Mode()&0o111 == 0 {
		return fmt.Errorf("local check %s is not an executable file; run chmod +x %s", shown, shown)
	}
	return nil
}
