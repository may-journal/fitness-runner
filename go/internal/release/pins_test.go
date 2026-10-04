package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pinFixture(t *testing.T) Config {
	t.Helper()
	c := testConfig(t)
	for _, path := range PinFiles() {
		text := "uses: may-journal/fitness-runner@go/v0.20261004.951\n"
		switch path {
		case "action.yml", "docs/ci.md":
			text += "https://github.com/may-journal/fitness-runner/releases/download/go/v0.20261004.1140/fitness-install-0.20261004.1140-linux-amd64\n"
		case "go/cmd/fitness-install/main.go":
			text = "package main\nvar version = \"0.20261004.1140\"\n"
		case "README.md":
			text = "Pin a version with `@v0.20260719.852`.\n"
		}
		testWrite(t, filepath.Join(c.Root, path), text+"uses: other/tool@go/v0.20261004.951\n")
	}
	return c
}

func TestUpdatePinsAndRepeat(t *testing.T) {
	c := pinFixture(t)
	changed, err := c.UpdatePins("0.20261004.1200")
	if err != nil || len(changed) != len(PinFiles()) {
		t.Fatalf("changed %v: %v", changed, err)
	}
	for _, path := range changed {
		assertUpdatedPin(t, c, path)
	}
	assertPinRepeat(t, c)
}

func assertPinRepeat(t *testing.T, c Config) {
	t.Helper()
	changed, err := c.UpdatePins("0.20261004.1200")
	if err != nil || len(changed) != 0 {
		t.Fatalf("repeat changed %v: %v", changed, err)
	}
}

func assertUpdatedPin(t *testing.T, c Config, path string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(c.Root, path))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "0.20261004.1200") || strings.Contains(text, "0.20261004.1140") {
		t.Fatalf("missing upgrade in %s: %s", path, text)
	}
	if !strings.Contains(text, "other/tool@go/v0.20261004.951") {
		t.Fatalf("unrelated dependency changed in %s", path)
	}
}

func TestRejectDowngradeBeforeWriting(t *testing.T) {
	c := pinFixture(t)
	path := filepath.Join(c.Root, "action.yml")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	testWrite(t, filepath.Join(c.Root, "README.md"), "Pin a version with `@v0.20261005.1`.\n")
	_, err = c.UpdatePins("0.20261004.1200")
	if err == nil || !strings.Contains(err.Error(), "downgrade README.md") {
		t.Fatalf("expected stale release rejection: %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(before) {
		t.Fatal("partially wrote stale release")
	}
}

func TestPinVersionValidation(t *testing.T) {
	c := pinFixture(t)
	for _, version := range []string{"", "go/v0.20261004.1", "v1.2.3", "1.2", "1.2.3-beta", "01.2.3", "1.2.03", "1.2.3\n"} {
		_, err := c.UpdatePins(version)
		requireError(t, err)
	}
}

func TestPinVersionComparison(t *testing.T) {
	pairs := [][2]string{{"0.20261004.951", "0.20261004.1140"}, {"0.20261004.1140", "0.20261005.1"}, {"9.99.99", "10.0.0"}, {"1.2.3", "1.2.99999999999999999999999999"}}
	for _, pair := range pairs {
		if comparePinVersions(pair[0], pair[1]) >= 0 || comparePinVersions(pair[1], pair[0]) <= 0 {
			t.Fatalf("wrong numeric ordering: %v", pair)
		}
	}
	if comparePinVersions("1.2.3", "1.2.3") != 0 {
		t.Fatal("equal versions compare unequal")
	}
}

func TestNormalizeOnlyFitnessPins(t *testing.T) {
	before := "uses: may-journal/fitness-runner@go/v0.20261004.951\nuses: other/tool@go/v0.20261004.951\n"
	after := strings.Replace(before, "fitness-runner@go/v0.20261004.951", "fitness-runner@go/v0.20261004.1140", 1)
	path := ".github/workflows/ci-reusable.yml"
	if NormalizePins(path, before) != NormalizePins(path, after) {
		t.Fatal("pin-only edit not normalized")
	}
	changedCode := after + "run: unexpected-command\n"
	if NormalizePins(path, before) == NormalizePins(path, changedCode) {
		t.Fatal("code change masked")
	}
	if NormalizePins("unrelated.yml", before) != before {
		t.Fatal("unknown file changed")
	}
}

func TestPinReadFailureDoesNotWrite(t *testing.T) {
	c := pinFixture(t)
	if err := os.Remove(filepath.Join(c.Root, "README.md")); err != nil {
		t.Fatal(err)
	}
	_, err := c.UpdatePins("0.20261004.1200")
	requireError(t, err)
	data, _ := os.ReadFile(filepath.Join(c.Root, "action.yml"))
	if strings.Contains(string(data), "1200") {
		t.Fatal("partially wrote after read failure")
	}
}
