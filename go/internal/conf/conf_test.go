package conf

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingIsNil(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if cfg != nil || err != nil {
		t.Fatalf("got %+v, %v", cfg, err)
	}
}

// writeConfigFile writes content to dir/name, failing the test on error.
func writeConfigFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadParsesKeys(t *testing.T) {
	t.Run("JSON wins over legacy", testLoadJSONWinsOverLegacy)
	dir := t.TempDir()
	content := `{
  "checks": ["changelog"],
  "disabledChecks": ["cspell"],
  "repeatedStringLiterals": {"allow": ["dist"]}
}`
	writeConfigFile(t, dir, FileName, content)
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.LegacyChecks) != 1 || cfg.DisabledChecks[0] != "cspell" {
		t.Fatalf("cfg: %+v", cfg)
	}
	if cfg.RepeatedStringLiterals.Allow[0] != "dist" {
		t.Fatalf("allow: %v", cfg.RepeatedStringLiterals.Allow)
	}
}

func TestLoadLegacyConfigErrors(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, dir, ".fitnessrc.js", "module.exports={}")
	_, err := Load(dir)
	if !errors.Is(err, ErrLegacyConfig) {
		t.Fatalf("want ErrLegacyConfig, got %v", err)
	}
}

func testLoadJSONWinsOverLegacy(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, dir, ".fitnessrc.js", "x")
	writeConfigFile(t, dir, FileName, `{"checks":["a"]}`)
	cfg, err := Load(dir)
	if err != nil || cfg == nil || cfg.LegacyChecks[0] != "a" {
		t.Fatalf("got %+v, %v", cfg, err)
	}
}

func TestLoadInvalidJSONErrors(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, dir, FileName, "{nope")
	if _, err := Load(dir); err == nil {
		t.Fatal("invalid JSON must error")
	}
}

func TestLoadRejectsRetiredKeys(t *testing.T) {
	cases := []struct {
		name, content, want string
	}{
		{"ignore", `{"ignore": ["x"]}`, `sets "ignore"; `},
		{"skip dirs", `{"skipTheseDirectories": []}`, `sets "skipTheseDirectories"; `},
		{"exempt", `{"proseBudget": {"maxWords": 9, "exempt": ["a.md"]}}`, `sets "proseBudget.exempt"; `},
		{"all three", `{"ignore": [], "skipTheseDirectories": [], "proseBudget": {"exempt": []}}`,
			`sets "ignore", "skipTheseDirectories", "proseBudget.exempt"; `},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, FileName), []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(dir)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}
