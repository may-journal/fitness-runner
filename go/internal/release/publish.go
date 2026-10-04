package release

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

func (c Config) VerifyTag(tag string) error {
	version, err := c.Version()
	if err != nil {
		return err
	}
	if tag != "go/v"+version {
		return fmt.Errorf("tag %s does not match changelog version go/v%s", tag, version)
	}
	return nil
}

func Notes(changelog string) string {
	start := strings.Index(changelog, "\n### ")
	if start < 0 {
		return ""
	}
	text := changelog[start+1:]
	if end := strings.Index(text, "\n### "); end >= 0 {
		text = text[:end]
	}
	return text
}

func (c Config) Publish(ctx context.Context) error {
	metadata, err := c.Metadata()
	if err != nil {
		return err
	}
	if err := c.VerifyTag(os.Getenv("GITHUB_REF_NAME")); err != nil {
		return err
	}
	path, err := c.releaseNotes()
	if err != nil {
		return err
	}
	return c.publishAssets(ctx, metadata, path)
}

func (c Config) releaseNotes() (string, error) {
	data, err := os.ReadFile(filepath.Join(c.Root, "CHANGELOG.md"))
	if err != nil {
		return "", err
	}
	path := filepath.Join(c.Out, "notes.md")
	return path, os.WriteFile(path, []byte(Notes(string(data))), 0600)
}

func (c Config) publishAssets(ctx context.Context, metadata Metadata, notes string) error {
	args := []string{"release", "create", "go/v" + metadata.Version}
	for _, asset := range metadata.Assets {
		args = append(args, filepath.Join(c.Out, asset.Name))
	}
	args = append(args, filepath.Join(c.Out, "checksums.txt"), "--verify-tag", "--title", "fitness v"+metadata.Version, "--notes-file", notes, "--latest=false")
	if err := c.runPublishCommand(ctx, args...); err != nil {
		return c.resumePublish(ctx, metadata, err)
	}
	return nil
}

// Retrying a publication never replaces bytes that a consumer may have cached.
func (c Config) resumePublish(ctx context.Context, metadata Metadata, createErr error) error {
	tag := "go/v" + metadata.Version
	data, err := c.publishOutput(ctx, "release", "view", tag, "--json", "isDraft,isPrerelease,assets")
	if err != nil {
		return fmt.Errorf("create release: %w; inspect existing release: %v", createErr, err)
	}
	names, err := publishedAssetNames(data)
	if err != nil {
		return err
	}
	return c.resumeAssets(ctx, tag, metadata, names)
}

func publishedAssetNames(data []byte) ([]string, error) {
	var release struct {
		IsDraft      bool
		IsPrerelease bool
		Assets       []struct{ Name string }
	}
	if err := json.Unmarshal(data, &release); err != nil {
		return nil, err
	}
	if release.IsDraft || release.IsPrerelease {
		return nil, fmt.Errorf("existing release must be published and stable")
	}
	var names []string
	for _, asset := range release.Assets {
		names = append(names, asset.Name)
	}
	return names, nil
}

func (c Config) resumeAssets(ctx context.Context, tag string, metadata Metadata, names []string) error {
	assets := append(slices.Clone(metadata.Assets), Asset{Name: "checksums.txt"})
	// Validate all existing assets before adding anything to a partial release.
	for _, asset := range assets {
		if err := c.checkPublishedAsset(ctx, tag, asset.Name, names); err != nil {
			return err
		}
	}
	return c.uploadMissingAssets(ctx, tag, assets, names)
}

func (c Config) checkPublishedAsset(ctx context.Context, tag, name string, names []string) error {
	if !slices.Contains(names, name) {
		return nil
	}
	directory, err := os.MkdirTemp("", "fitness-published-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	if err := c.runPublishCommand(ctx, "release", "download", tag, "--pattern", name, "--dir", directory); err != nil {
		return err
	}
	return comparePublishedFile(filepath.Join(c.Out, name), filepath.Join(directory, name))
}

func comparePublishedFile(local, published string) error {
	expected, err := fileHash(local)
	if err != nil {
		return err
	}
	actual, err := fileHash(published)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("published asset %s differs; refusing to replace it", filepath.Base(local))
	}
	return nil
}

func (c Config) uploadMissingAssets(ctx context.Context, tag string, assets []Asset, names []string) error {
	for _, asset := range assets {
		if slices.Contains(names, asset.Name) {
			continue
		}
		if err := c.runPublishCommand(ctx, "release", "upload", tag, filepath.Join(c.Out, asset.Name)); err != nil {
			return err
		}
	}
	return nil
}

func (c Config) runPublishCommand(ctx context.Context, args ...string) error {
	command := exec.CommandContext(ctx, "gh", args...)
	command.Dir = c.Root
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	return command.Run()
}

func (c Config) publishOutput(ctx context.Context, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "gh", args...)
	command.Dir = c.Root
	command.Stderr = os.Stderr
	return command.Output()
}

// Promote marks a verified release latest without downgrading a newer release.
// The workflow serializes this read and write with other promotion jobs.
func (c Config) Promote(ctx context.Context) error {
	metadata, err := c.Metadata()
	if err != nil {
		return err
	}
	if !pinVersionPattern.MatchString(metadata.Version) {
		return fmt.Errorf("invalid release version %q", metadata.Version)
	}
	latest, err := c.latestPublishedVersion(ctx)
	if err != nil {
		return err
	}
	if comparePinVersions(latest, metadata.Version) >= 0 {
		return nil
	}
	return c.runPublishCommand(ctx, "release", "edit", "go/v"+metadata.Version, "--latest")
}

func (c Config) latestPublishedVersion(ctx context.Context) (string, error) {
	data, err := c.publishOutput(ctx, "release", "view", "--json", "tagName")
	if err != nil {
		return "", err
	}
	var release struct{ TagName string }
	if err := json.Unmarshal(data, &release); err != nil {
		return "", err
	}
	version := strings.TrimPrefix(release.TagName, "go/v")
	if !pinVersionPattern.MatchString(version) {
		return "", fmt.Errorf("invalid latest release tag %q", release.TagName)
	}
	return version, nil
}
