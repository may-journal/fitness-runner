package release

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const externalConfig = `{"policy":"external"}`

// checkHookTools proves a direct installer call reaches the bundled stamper
// without consumers resolving the cache directory or adding it to PATH.
func (s smoke) checkHookTools(ctx context.Context) error {
	config := filepath.Join(s.repo, ".fitnessrc.json")
	if err := s.stageHookFixture(ctx, config); err != nil {
		return err
	}
	if err := s.run(ctx, s.installer, "--", "hook", "pre-commit"); err != nil {
		return err
	}
	data, err := s.command(ctx, "git", "show", ":CHANGELOG.md").Output()
	if err != nil {
		return err
	}
	if err := validateHookStamp(string(data)); err != nil {
		return err
	}
	return os.WriteFile(config, []byte(externalConfig), 0600)
}

func validateHookStamp(data string) error {
	if strings.Contains(data, "2000.01.01.0000") || !strings.Contains(data, "- Chore: verify bundled hook tools.") {
		return fmt.Errorf("bundled stamper did not update the staged changelog: %s", data)
	}
	return nil
}

func (s smoke) stageHookFixture(ctx context.Context, config string) error {
	if err := os.WriteFile(config, []byte(`{"policy":"external","checks":["changelog"]}`), 0600); err != nil {
		return err
	}
	const notes = "# Changelog\n\n## Changes\n\n### 2000.01.01.0000\n\n- Chore: verify bundled hook tools.\n"
	if err := os.WriteFile(filepath.Join(s.repo, "CHANGELOG.md"), []byte(notes), 0600); err != nil {
		return err
	}
	return s.run(ctx, "git", "add", "CHANGELOG.md")
}
