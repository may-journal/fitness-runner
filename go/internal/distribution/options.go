// Package distribution installs verified release bundles without a compiler.
package distribution

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const releaseRoot = "https://github.com/may-journal/fitness-runner/releases"
const latestRelease = "https://api.github.com/repos/may-journal/fitness-runner/releases/latest"
const checksumName = "checksums.txt"

var versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
var hashPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var binaryPattern = regexp.MustCompile(`^fitness(-[a-z0-9-]+)?$`)

// Options separates installer flags from untouched runner arguments.
type Options struct {
	Version     string
	InstallOnly bool
	Args        []string
}

func ParseOptions(args []string, version string, output io.Writer) (Options, error) {
	var options Options
	flags := flag.NewFlagSet("fitness-install", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.StringVar(&options.Version, "version", version, "exact release version, or explicit latest")
	flags.BoolVar(&options.InstallOnly, "install-only", false, "print the installed binary directory without running checks")
	if err := flags.Parse(args); err != nil {
		return options, err
	}
	options.Args = flags.Args()
	if options.InstallOnly && len(options.Args) > 0 {
		return options, errors.New("install-only does not accept runner arguments")
	}
	return options, nil
}

func normalizeVersion(version string) (string, error) {
	version = strings.TrimPrefix(strings.TrimPrefix(version, "go/"), "v")
	if !versionPattern.MatchString(version) {
		return "", fmt.Errorf("invalid release version %q", version)
	}
	return version, nil
}

func Platform(system, arch string) (string, error) {
	supported := map[string]bool{"linux-amd64": true, "linux-arm64": true, "darwin-amd64": true, "darwin-arm64": true}
	platform := system + "-" + arch
	if !supported[platform] {
		return "", fmt.Errorf("unsupported platform %s; use Linux or macOS on amd64 or arm64", platform)
	}
	return platform, nil
}

func cacheDirectory() (string, error) {
	if path := os.Getenv("FITNESS_CACHE_DIR"); path != "" {
		return filepath.Abs(path)
	}
	if path := os.Getenv("XDG_CACHE_HOME"); path != "" {
		return filepath.Abs(filepath.Join(path, "fitness"))
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "fitness"), nil
}

// ParseHashes reads release-time platform=hash pairs injected by the linker.
func ParseHashes(encoded string) map[string]string {
	hashes := map[string]string{}
	for _, pair := range strings.Split(encoded, ",") {
		key, value, ok := strings.Cut(pair, "=")
		if ok {
			hashes[key] = value
		}
	}
	return hashes
}
