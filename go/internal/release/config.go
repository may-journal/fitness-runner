// Package release verifies GoReleaser assets and prepares installer hash metadata.
package release

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const metadataName = "release.json"
const runnerName = "fitness"
const installerName = "fitness-install"
const verifierName = "fitness-release"

var platforms = []string{"darwin-amd64", "darwin-arm64", "linux-amd64", "linux-arm64"}

type Asset struct {
	Name string `json:"name"`
	Hash string `json:"sha256"`
}
type Metadata struct {
	Version string  `json:"version"`
	Assets  []Asset `json:"assets"`
}

// Config locates the release outputs. Platforms lists the platforms whose
// outputs bundle-hashes and assemble expect; New sets all four.
type Config struct {
	Root, Out string
	Platforms []string
}

func New(root string) (Config, error) {
	root, err := filepath.Abs(root)
	return Config{Root: root, Out: filepath.Join(root, "out"), Platforms: platforms}, err
}

// Host limits the expected outputs to the platform this tool runs on.
func (c Config) Host() Config {
	c.Platforms = []string{runtime.GOOS + "-" + runtime.GOARCH}
	return c
}

func (c Config) Version() (string, error) {
	data, err := os.ReadFile(filepath.Join(c.Root, "version.txt"))
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(data))
	if !versionPattern.MatchString(version) {
		return "", fmt.Errorf("invalid version.txt version %q", version)
	}
	return version, nil
}

func (c Config) Metadata() (Metadata, error) {
	var metadata Metadata
	data, err := os.ReadFile(filepath.Join(c.Out, metadataName))
	if err != nil {
		return metadata, err
	}
	err = json.Unmarshal(data, &metadata)
	return metadata, err
}

func (c Config) saveMetadata(metadata Metadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.Out, metadataName), data, 0600)
}

func fileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

func executable(name, version, platform string) string { return name + "-" + version + "-" + platform }
func bundleName(version, platform string) string {
	return executable(runnerName, version, platform) + ".tar.gz"
}
func (c Config) asset(name string) (Asset, error) {
	hash, err := fileHash(filepath.Join(c.Out, name))
	return Asset{Name: name, Hash: hash}, err
}

func encodedHashes(metadata Metadata) string {
	pairs := make([]string, 0, len(metadata.Assets))
	for _, asset := range metadata.Assets {
		platform := strings.TrimSuffix(strings.TrimPrefix(asset.Name, "fitness-"+metadata.Version+"-"), ".tar.gz")
		pairs = append(pairs, platform+"="+asset.Hash)
	}
	return strings.Join(pairs, ",")
}
