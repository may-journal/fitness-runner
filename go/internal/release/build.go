// cspell:ignore CGO GOOS GOARCH ldflags buildvcs trimpath
package release

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (c Config) Build(ctx context.Context) error {
	version, err := c.Version()
	if err != nil {
		return err
	}
	if err := os.RemoveAll(c.Out); err != nil {
		return err
	}
	if err := os.MkdirAll(c.Out, 0700); err != nil {
		return err
	}
	return c.buildRelease(ctx, version)
}

func (c Config) buildRelease(ctx context.Context, version string) error {
	metadata := Metadata{Version: version}
	for _, platform := range platforms {
		asset, err := c.buildBundle(ctx, version, platform)
		if err != nil {
			return err
		}
		metadata.Assets = append(metadata.Assets, asset)
	}
	return c.finishRelease(ctx, metadata)
}

func (c Config) finishRelease(ctx context.Context, metadata Metadata) error {
	hashes := encodedHashes(metadata)
	for _, platform := range platforms {
		asset, err := c.buildTools(ctx, metadata.Version, platform, hashes)
		if err != nil {
			return err
		}
		metadata.Assets = append(metadata.Assets, asset)
	}
	if err := c.saveMetadata(metadata); err != nil {
		return err
	}
	return c.writeChecksums(metadata)
}

func (c Config) compile(ctx context.Context, platform, output, flags, target string) error {
	parts := strings.Split(platform, "-")
	cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-buildvcs=false", "-ldflags", flags, "-o", output, target)
	cmd.Dir = filepath.Join(c.Root, "go")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+parts[0], "GOARCH="+parts[1])
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func (c Config) buildBundle(ctx context.Context, version, platform string) (Asset, error) {
	top := executable(runnerName, version, platform)
	directory := filepath.Join(c.Out, top)
	if err := os.Mkdir(directory, 0700); err != nil {
		return Asset{}, err
	}
	defer os.RemoveAll(directory)
	if err := c.compile(ctx, platform, directory, "", "./cmd/..."); err != nil {
		return Asset{}, err
	}
	if err := removeTools(directory); err != nil {
		return Asset{}, err
	}
	name := bundleName(version, platform)
	if err := archiveDirectory(filepath.Join(c.Out, name), directory); err != nil {
		return Asset{}, err
	}
	return c.asset(name)
}

func removeTools(directory string) error {
	for _, name := range []string{installerName, verifierName} {
		if err := os.Remove(filepath.Join(directory, name)); err != nil {
			return err
		}
	}
	return nil
}

func (c Config) buildTools(ctx context.Context, version, platform, hashes string) (Asset, error) {
	name := executable(installerName, version, platform)
	flags := "-X main.version=" + version + " -X main.bundleHashes=" + hashes
	if err := c.compile(ctx, platform, filepath.Join(c.Out, name), flags, "./cmd/fitness-install"); err != nil {
		return Asset{}, err
	}
	helper := filepath.Join(c.Out, verifierName+"-"+platform)
	if err := c.compile(ctx, platform, helper, "", "./cmd/fitness-release"); err != nil {
		return Asset{}, err
	}
	return c.asset(name)
}

func (c Config) writeChecksums(metadata Metadata) error {
	var text strings.Builder
	for _, asset := range metadata.Assets {
		fmt.Fprintf(&text, "%s  %s\n", asset.Hash, asset.Name)
	}
	return os.WriteFile(filepath.Join(c.Out, "checksums.txt"), []byte(text.String()), 0600)
}
