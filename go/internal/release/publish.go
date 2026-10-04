package release

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func (c Config) VerifyTag(tag string) error {
	version, err := c.Version()
	if err != nil {
		return err
	}
	if tag != "v"+version {
		return fmt.Errorf("tag %s does not match version.txt v%s", tag, version)
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
	return c.runPublishCommand(ctx, "release", "edit", "v"+metadata.Version, "--prerelease=false", "--latest")
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
	version := strings.TrimPrefix(strings.TrimPrefix(release.TagName, "go/"), "v")
	if !pinVersionPattern.MatchString(version) {
		return "", fmt.Errorf("invalid latest release tag %q", release.TagName)
	}
	return version, nil
}
