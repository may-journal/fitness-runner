package release

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func retryPublisher(t *testing.T, c Config, existing, content string) string {
	t.Helper()
	directory := t.TempDir()
	log := filepath.Join(directory, "calls")
	t.Setenv("RETRY_LOG", log)
	t.Setenv("RETRY_ASSETS", existing)
	t.Setenv("RETRY_CONTENT", content)
	t.Setenv("RETRY_LATEST", `{"tagName":"go/v0.20261004.36"}`)
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$RETRY_LOG"
case "$1 $2" in
 'release create') exit 1 ;;
 'release view')
  if [ "$3" = --json ]; then printf '%s' "$RETRY_LATEST"; else printf '%s' "$RETRY_ASSETS"; fi ;;
 'release download')
  if [ "$5" = checksums.txt ]; then printf manifest > "$7/$5"; else printf '%s' "$RETRY_CONTENT" > "$7/$5"; fi ;;
 'release upload'|'release edit') exit 0 ;;
 *) exit 2 ;;
esac
`
	path := filepath.Join(directory, "gh")
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	t.Setenv("GITHUB_REF_NAME", "go/v0.20261004.37")
	metadata := Metadata{Version: "0.20261004.37", Assets: []Asset{{Name: "bundle.tar.gz"}}}
	if err := c.saveMetadata(metadata); err != nil {
		t.Fatal(err)
	}
	testWrite(t, filepath.Join(c.Out, "bundle.tar.gz"), "bundle")
	testWrite(t, filepath.Join(c.Out, "checksums.txt"), "manifest")
	return log
}

func publishCalls(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestPublishResumesPartialRelease(t *testing.T) {
	c := testConfig(t)
	log := retryPublisher(t, c, `{"assets":[{"name":"bundle.tar.gz"}]}`, "bundle")
	if err := c.Publish(context.Background()); err != nil {
		t.Fatal(err)
	}
	calls := publishCalls(t, log)
	if !strings.Contains(calls, "release upload go/v0.20261004.37 "+filepath.Join(c.Out, "checksums.txt")) {
		t.Fatal(calls)
	}
	if strings.Contains(calls, "--clobber") {
		t.Fatal(calls)
	}
}

func TestPublishRejectsConflictingAssetBeforeUpload(t *testing.T) {
	c := testConfig(t)
	log := retryPublisher(t, c, `{"assets":[{"name":"bundle.tar.gz"}]}`, "wrong")
	err := c.Publish(context.Background())
	if err == nil || !strings.Contains(err.Error(), "refusing to replace") {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(publishCalls(t, log), "release upload") {
		t.Fatal("uploaded before rejecting conflicting asset")
	}
}

func TestPublishRejectsDraftAndPrerelease(t *testing.T) {
	for _, state := range []string{`{"isDraft":true}`, `{"isPrerelease":true}`, `invalid`} {
		t.Run(state, func(t *testing.T) {
			c := testConfig(t)
			log := retryPublisher(t, c, state, "")
			if err := c.Publish(context.Background()); err == nil {
				t.Fatal("accepted invalid release")
			}
			if strings.Contains(publishCalls(t, log), "release upload") {
				t.Fatal("uploaded to invalid release")
			}
		})
	}
}

func TestPromoteDoesNotDowngrade(t *testing.T) {
	for _, version := range []string{"0.20261004.38", "0.20261004.37", "0.20261004.36"} {
		t.Run(version, func(t *testing.T) {
			c := testConfig(t)
			log := retryPublisher(t, c, `{"assets":[]}`, "")
			t.Setenv("RETRY_LATEST", `{"tagName":"go/v`+version+`"}`)
			if err := c.Promote(context.Background()); err != nil {
				t.Fatal(err)
			}
			edited := strings.Contains(publishCalls(t, log), "release edit")
			if edited != (version == "0.20261004.36") {
				t.Fatalf("promotion=%v for %s", edited, version)
			}
		})
	}
}

func TestPromoteRejectsUnknownLatestTag(t *testing.T) {
	c := testConfig(t)
	log := retryPublisher(t, c, `{"assets":[]}`, "")
	t.Setenv("RETRY_LATEST", `{"tagName":"unexpected"}`)
	if err := c.Promote(context.Background()); err == nil {
		t.Fatal("accepted unknown latest tag")
	}
	if strings.Contains(publishCalls(t, log), "release edit") {
		t.Fatal("changed latest on invalid data")
	}
}

func TestPublishCompleteRetryPreservesAssets(t *testing.T) {
	c := testConfig(t)
	log := retryPublisher(t, c, `{"assets":[{"name":"bundle.tar.gz"},{"name":"checksums.txt"}]}`, "bundle")
	if err := c.Publish(context.Background()); err != nil {
		t.Fatal(err)
	}
	calls := publishCalls(t, log)
	if strings.Contains(calls, "release upload") {
		t.Fatal("complete release was changed: " + calls)
	}
	if strings.Count(calls, "release download") != 2 {
		t.Fatal("did not verify every asset: " + calls)
	}
}
