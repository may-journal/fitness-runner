package release

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PreparePins changes files for the pull-request action; it never changes Git refs.
func (c Config) PreparePins() error {
	metadata, err := c.Metadata()
	if err != nil {
		return err
	}
	if !pinVersionPattern.MatchString(metadata.Version) {
		return fmt.Errorf("invalid release version %q", metadata.Version)
	}
	return c.prepareVersionPins(metadata.Version)
}

func (c Config) prepareVersionPins(version string) error {
	edits, err := c.preparePinEdits(version)
	if err != nil {
		return err
	}
	if len(edits) == 0 {
		return pinOutputs(version, false)
	}
	changelog, err := c.pinChangelogEdit(version)
	if err != nil {
		return err
	}
	if _, err := c.writePinEdits(append(edits, changelog)); err != nil {
		return err
	}
	return pinOutputs(version, true)
}

func pinNotes(version string) string {
	return "- Chore: update Fitness pins to verified release v" + version + ".\n- Chore: keep the installer and shared workflows on the same release.\n- Docs: refresh the pinned release used in setup examples.\n"
}

func (c Config) pinChangelogEdit(version string) (pinEdit, error) {
	data, err := os.ReadFile(filepath.Join(c.Root, "CHANGELOG.md"))
	if err != nil {
		return pinEdit{}, err
	}
	const marker = "\n## Changes\n\n"
	if !strings.Contains(string(data), marker) {
		return pinEdit{}, fmt.Errorf("CHANGELOG.md has no Changes section")
	}
	section := "### " + time.Now().UTC().Format("2006.01.02.1504") + "\n\n" + pinNotes(version) + "\n"
	return pinEdit{path: "CHANGELOG.md", text: strings.Replace(string(data), marker, marker+section, 1)}, nil
}

func pinOutputs(version string, changed bool) error {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		return nil
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	text := "version=" + version + "\n"
	if changed {
		text += "changed=true\n"
	}
	_, err = file.WriteString(text)
	return err
}
