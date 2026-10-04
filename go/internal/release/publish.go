package release

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	args = append(args, filepath.Join(c.Out, "checksums.txt"), "--verify-tag", "--title", "fitness v"+metadata.Version, "--notes-file", notes)
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = c.Root
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}
