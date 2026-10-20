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

// skillsFolder holds one folder per skill beside the binaries in a bundle. A skill
// file may sit in subfolders and none may start with a dot, so no path escapes.
const skillsFolder = "skills"

var skillFolderPattern = regexp.MustCompile(`^skills/[a-z0-9]+(-[a-z0-9]+)*$`)
var skillPattern = regexp.MustCompile(`^skills/[a-z0-9]+(-[a-z0-9]+)*(/[A-Za-z0-9_-][A-Za-z0-9._-]*)+$`)

// Options separates installer flags from untouched runner arguments.
type Options struct {
	Version      string
	InstallOnly  bool
	GitHubAction bool
	InstallHook  bool
	LinkSkills   []string
	Args         []string
}

func ParseOptions(args []string, version string, output io.Writer) (Options, error) {
	var options Options
	flags := flag.NewFlagSet("fitness-install", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.StringVar(&options.Version, "version", version, "exact release version, or explicit latest")
	flags.BoolVar(&options.InstallHook, "install-hook", false, "install a Go pre-commit hook without replacing existing hooks")
	flags.BoolVar(&options.GitHubAction, "github-action", false, "read action inputs and publish action outputs")
	flags.BoolVar(&options.InstallOnly, "install-only", false, "print the installed binary directory without running checks")
	flags.Func("link-skills", "link the release skills into this folder, repeatable, without running checks", func(folder string) error {
		options.LinkSkills = append(options.LinkSkills, folder)
		return nil
	})
	if err := flags.Parse(args); err != nil {
		return options, err
	}
	options.Args = flags.Args()
	return options, options.validate()

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

func (options Options) validate() error {
	if options.InstallOnly && len(options.Args) > 0 {
		return errors.New("install-only does not accept runner arguments")
	}
	if err := options.validateLinkSkills(); err != nil {
		return err
	}
	if options.InstallHook {
		return options.validateHook()
	}
	return nil
}

func (options Options) validateLinkSkills() error {
	if len(options.LinkSkills) == 0 {
		return nil
	}
	if len(options.Args) > 0 || options.otherMode() {
		return errors.New("link-skills does not combine with other modes or runner arguments")
	}
	return nil
}

func (options Options) otherMode() bool {
	return options.GitHubAction || options.InstallOnly || options.InstallHook
}

func (options Options) validateHook() error {
	if options.InstallOnly || options.GitHubAction {
		return errors.New("install-hook cannot be combined with install-only or github-action")
	}
	return nil
}
