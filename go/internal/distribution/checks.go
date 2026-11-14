// cspell:ignore Typeflag
package distribution

import (
	"archive/tar"
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/pkg/par"
)

// checksRoot is where fitness-checks publishes one release per check and
// version, each with one tar and one SHA-256 file per platform.
const checksRoot = "https://github.com/may-journal/fitness-checks/releases"

// checkPrefix starts every check program's name and every check release tag.
const checkPrefix = "fitness-check-"

// maxCheckBytes bounds one check archive and the program inside it.
const maxCheckBytes = 128 << 20

// checkWorkers is how many check archives download side by side.
const checkWorkers = 8

// CheckPin is one check a repo installs and its pin: a major such as v1, or
// an exact release such as v1.2.3.
type CheckPin struct {
	Name, Pin string
}

// ParsePins reads the `<name> <pin>` lines `fitness check-versions` prints.
func ParsePins(text string) ([]CheckPin, error) {
	var pins []CheckPin
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("fitness check-versions printed %q, not `<name> <pin>`", line)
		}
		pins = append(pins, CheckPin{Name: fields[0], Pin: fields[1]})
	}
	return pins, nil
}

// InstallChecks installs each pinned check from its verified release and
// returns a folder that holds exactly those programs, for FITNESS_CHECKS_DIR.
func (i *Installer) InstallChecks(ctx context.Context, pins []CheckPin) (string, error) {
	if i.Unanswered {
		return "", errors.New("GitHub did not answer the release lookup, so no check installs")
	}
	releases, err := i.resolvePins(ctx, pins)
	if err != nil {
		return "", err
	}
	paths, err := par.MapErr(ctx, len(releases), checkWorkers, func(ctx context.Context, k int) (string, error) {
		return i.installCheck(ctx, pins[k].Name, releases[k])
	})
	if err != nil {
		return "", err
	}
	dir, err := checkSet(i.Cache, paths)
	if err == nil {
		fmt.Fprintf(i.Log, "fitness: installed %d checks in %s\n", len(paths), dir)
	}
	return dir, err
}

// installCheck returns the cached program of one check release, or downloads,
// verifies and caches it.
func (i *Installer) installCheck(ctx context.Context, name string, release checkRelease) (string, error) {
	asset := release.Tag + "-" + i.Platform + ".tar"
	program := filepath.Join(i.Cache, "checks", asset, checkPrefix+name)
	if _, err := os.Stat(program); err == nil {
		return program, nil
	}
	data, err := i.verifiedCheck(ctx, release, asset)
	if err != nil {
		return "", err
	}
	binary, err := checkProgram(data, checkPrefix+name)
	if err != nil {
		return "", fmt.Errorf("%s: %w", asset, err)
	}
	return program, writeProgram(program, binary)
}

// verifiedCheck downloads one check archive, failing unless it matches the
// SHA-256 the index gave or the release's own SHA-256 file holds.
func (i *Installer) verifiedCheck(ctx context.Context, release checkRelease, asset string) ([]byte, error) {
	url := cmp.Or(release.URL, checksRoot+"/download/"+release.Tag+"/"+asset)
	want, err := i.checkHash(ctx, release.SHA256, url, asset)
	if err != nil {
		return nil, err
	}
	data, err := i.download(ctx, url, asset)
	if err == nil && contentHash(data) != want {
		err = fmt.Errorf("checksum mismatch for %s", asset)
	}
	return data, err
}

// checkHash is the index's SHA-256, else the one the release's SHA-256 file
// beside url holds.
func (i *Installer) checkHash(ctx context.Context, indexed, url, asset string) (string, error) {
	if indexed != "" {
		return checkedHash(indexed)
	}
	sum, err := i.fetch(ctx, asset+".sha256", url+".sha256", maxMetadataBytes)
	if err != nil {
		return "", err
	}
	hash, err := checkedHash(strings.Fields(string(sum) + " ")[0])
	if err != nil {
		return "", fmt.Errorf("%s.sha256: %w", asset, err)
	}
	return hash, nil
}

// checkProgram returns the one regular file named name in a check archive.
func checkProgram(data []byte, name string) ([]byte, error) {
	reader := tar.NewReader(bytes.NewReader(data))
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("holds no %s program", name)
		}
		if err != nil {
			return nil, err
		}
		if isProgram(header, name) {
			return readBounded(reader, maxCheckBytes)
		}
	}
}

// isProgram reports whether header is the regular file name.
func isProgram(header *tar.Header, name string) bool {
	return header.Typeflag == tar.TypeReg && header.Name == name
}

// writeProgram writes an executable program into its own cache folder,
// renaming it into place so a parallel install never sees half a file.
func writeProgram(path string, binary []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if err := writeAtomic(path, binary); err != nil {
		return err
	}
	return os.Chmod(path, 0755)
}

// checkSet returns a cache folder linking exactly programs. Its name hashes
// their paths, so each selection of checks and versions gets its own.
func checkSet(cache string, programs []string) (string, error) {
	sorted := slices.Sorted(slices.Values(programs))
	dir := filepath.Join(cache, "check-sets", contentHash([]byte(strings.Join(sorted, "\n"))))
	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	}
	staged, err := os.MkdirTemp(filepath.Join(cache), "check-set-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(staged)
	if err := linkAll(staged, sorted); err != nil {
		return "", err
	}
	return dir, publishSet(staged, dir)
}

// linkAll links each program into dir under its own name.
func linkAll(dir string, programs []string) error {
	for _, program := range programs {
		if err := os.Symlink(program, filepath.Join(dir, filepath.Base(program))); err != nil {
			return err
		}
	}
	return nil
}

// publishSet renames a staged set into place, keeping one another install
// published first.
func publishSet(staged, dir string) error {
	if err := os.MkdirAll(filepath.Dir(dir), 0700); err != nil {
		return err
	}
	if err := os.Rename(staged, dir); err != nil && !exists(dir) {
		return err
	}
	return nil
}

// exists reports whether path exists.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
