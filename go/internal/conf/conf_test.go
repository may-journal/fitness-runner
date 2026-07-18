package conf

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingIsNil(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if cfg != nil || err != nil {
		t.Fatalf("got %+v, %v", cfg, err)
	}
}

func TestLoadParsesKeys(t *testing.T) {
	dir := t.TempDir()
	content := `{
  "checks": ["changelog", "./local/check"],
  "disabledChecks": ["cspell"],
  "skipTheseDirectories": ["vendor"],
  "repeatedStringLiterals": {"allow": ["dist"]}
}`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Checks) != 2 || cfg.Checks[1] != "./local/check" {
		t.Fatalf("checks: %v", cfg.Checks)
	}
	if cfg.DisabledChecks[0] != "cspell" || cfg.SkipTheseDirectories[0] != "vendor" {
		t.Fatalf("cfg: %+v", cfg)
	}
	if cfg.RepeatedStringLiterals.Allow[0] != "dist" {
		t.Fatalf("allow: %v", cfg.RepeatedStringLiterals.Allow)
	}
}

func TestLoadLegacyConfigErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fitnessrc.js"), []byte("module.exports={}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if !errors.Is(err, ErrLegacyConfig) {
		t.Fatalf("want ErrLegacyConfig, got %v", err)
	}
}

func TestLoadJSONWinsOverLegacy(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fitnessrc.js"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(`{"checks":["a"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil || cfg == nil || cfg.Checks[0] != "a" {
		t.Fatalf("got %+v, %v", cfg, err)
	}
}

func TestLoadInvalidJSONErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("invalid JSON must error")
	}
}
