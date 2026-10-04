package release

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BundleHashes verifies the first GoReleaser pass before its hashes are embedded
// by the installer build. GoReleaser owns archive and checksum generation.
func (c Config) BundleHashes() (string, error) {
	version, err := c.Version()
	if err != nil {
		return "", err
	}
	assets, err := c.readAssets("bundles", version, false)
	if err != nil {
		return "", err
	}
	return encodedHashes(Metadata{Version: version, Assets: assets}), nil
}

// Assemble prepares a flat consumer fixture from GoReleaser outputs. It neither
// compiles binaries nor creates archives, checksums, tags, or releases.
func (c Config) Assemble() error {
	version, err := c.Version()
	if err != nil {
		return err
	}
	assets, err := c.readAssets("tools", version, true)
	if err != nil {
		return err
	}
	if err := c.copyOutputs(assets); err != nil {
		return err
	}
	return c.saveMetadata(Metadata{Version: version, Assets: assets})
}

func (c Config) readAssets(stage, version string, installers bool) ([]Asset, error) {
	sums, err := readChecksums(filepath.Join(c.Out, stage, "checksums.txt"))
	if err != nil {
		return nil, err
	}
	var assets []Asset
	for _, name := range assetNames(version, installers) {
		asset, err := c.verifiedOutput(name, sums)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

func assetNames(version string, installers bool) []string {
	var names []string
	for _, platform := range platforms {
		names = append(names, bundleName(version, platform))
		if installers {
			names = append(names, executable(installerName, version, platform))
		}
	}
	return names
}

func (c Config) verifiedOutput(name string, sums map[string]string) (Asset, error) {
	expected, ok := sums[name]
	if !ok {
		return Asset{}, fmt.Errorf("GoReleaser checksums omit %s", name)
	}
	actual, err := fileHash(c.outputPath(name))
	if err != nil {
		return Asset{}, err
	}
	if actual != expected {
		return Asset{}, fmt.Errorf("GoReleaser checksum mismatch for %s", name)
	}
	return Asset{Name: name, Hash: actual}, nil
}

func (c Config) outputPath(name string) string {
	stage := "tools"
	if strings.HasSuffix(name, ".tar.gz") {
		stage = "bundles"
	}
	return filepath.Join(c.Out, stage, name)
}

func readChecksums(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	sums := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if err := readChecksumLine(line, sums); err != nil {
			return nil, err
		}
	}
	return sums, nil
}

func readChecksumLine(line string, sums map[string]string) error {
	fields := strings.Fields(line)
	if len(fields) != 2 {
		return fmt.Errorf("invalid GoReleaser checksum line %q", line)
	}
	hash, err := hex.DecodeString(fields[0])
	if err != nil || len(hash) != 32 {
		return fmt.Errorf("invalid SHA-256 checksum for %s", fields[1])
	}
	if _, exists := sums[fields[1]]; exists {
		return fmt.Errorf("duplicate checksum for %s", fields[1])
	}
	sums[fields[1]] = fields[0]
	return nil
}

func (c Config) copyOutputs(assets []Asset) error {
	for _, asset := range assets {
		if err := copyReleaseFile(c.outputPath(asset.Name), filepath.Join(c.Out, asset.Name), 0700); err != nil {
			return err
		}
	}
	if err := c.copyVerifiers(); err != nil {
		return err
	}
	return copyReleaseFile(filepath.Join(c.Out, "tools", "checksums.txt"), filepath.Join(c.Out, "checksums.txt"), 0600)
}

func (c Config) copyVerifiers() error {
	for _, platform := range platforms {
		name := verifierName + "-" + platform
		if err := copyReleaseFile(c.outputPath(name), filepath.Join(c.Out, name), 0700); err != nil {
			return err
		}
	}
	return nil
}

func copyReleaseFile(source, target string, mode os.FileMode) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(target, data, mode)
}
