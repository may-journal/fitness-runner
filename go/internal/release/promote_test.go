package release

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func promotionFixture(t *testing.T, tag string) (Config, string) {
	t.Helper()
	c := testConfig(t)
	if err := c.saveMetadata(Metadata{Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	log := filepath.Join(directory, "calls")
	t.Setenv("PROMOTION_LOG", log)
	t.Setenv("PROMOTION_LATEST", `{"tagName":"`+tag+`"}`)
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$PROMOTION_LOG"
case "$1 $2" in
 'release view') printf '%s' "$PROMOTION_LATEST" ;;
 'release edit') exit 0 ;;
 *) exit 2 ;;
esac
`
	if err := os.WriteFile(filepath.Join(directory, "gh"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	return c, log
}

func TestPromotePreservesNewestVerifiedRelease(t *testing.T) {
	for _, tag := range []string{"go/v0.20261004.1140", "v1.0.0", "v1.0.1"} {
		assertPromotion(t, tag)
	}
}

func assertPromotion(t *testing.T, tag string) {
	t.Helper()
	c, log := promotionFixture(t, tag)
	if err := c.Promote(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	promoted := strings.Contains(string(data), "release edit v1.0.0 --prerelease=false --latest")
	if promoted != (tag == "go/v0.20261004.1140") {
		t.Fatalf("wrong promotion: %s", data)
	}
}

func TestPromoteRejectsUnknownLatestVersion(t *testing.T) {
	c, log := promotionFixture(t, "unexpected")
	requireError(t, c.Promote(context.Background()))
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "release edit") {
		t.Fatal("edited after invalid version")
	}
}
