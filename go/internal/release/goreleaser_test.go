// cspell:ignore goreleaser
package release

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func goreleaserFixture(t *testing.T) Config {
	t.Helper()
	c := testConfig(t)
	var bundles, all strings.Builder
	for _, name := range assetNames("1.0.0", true) {
		testWrite(t, c.outputPath(name), "release payload "+name)
		hash, err := fileHash(c.outputPath(name))
		if err != nil {
			t.Fatal(err)
		}
		line := fmt.Sprintf("%s  %s\n", hash, name)
		all.WriteString(line)
		if strings.HasSuffix(name, ".tar.gz") {
			bundles.WriteString(line)
		}
	}
	testWrite(t, filepath.Join(c.Out, "bundles", "checksums.txt"), bundles.String())
	testWrite(t, filepath.Join(c.Out, "tools", "checksums.txt"), all.String())
	for _, platform := range platforms {
		testWrite(t, c.outputPath(verifierName+"-"+platform), "verifier")
	}
	return c
}

func TestGoReleaserAdapterPreservesVerifiedAssets(t *testing.T) {
	c := goreleaserFixture(t)
	hashes, err := c.BundleHashes()
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.Split(hashes, ",")) != 4 {
		t.Fatalf("hash map: %s", hashes)
	}
	if err := c.Assemble(); err != nil {
		t.Fatal(err)
	}
	assertGoReleaserManifest(t, c)
}

func assertGoReleaserManifest(t *testing.T, c Config) {
	t.Helper()
	metadata, err := c.Metadata()
	if err != nil || metadata.Version != "1.0.0" || len(metadata.Assets) != 8 {
		t.Fatalf("metadata: %+v %v", metadata, err)
	}
	assertAssembledAssets(t, c, metadata.Assets)
	assertChecksumCopy(t, c)
}

func assertAssembledAssets(t *testing.T, c Config, assets []Asset) {
	t.Helper()
	for _, asset := range assets {
		got, err := c.asset(asset.Name)
		if err != nil || got != asset {
			t.Fatalf("changed asset: %+v %v", got, err)
		}
	}
}

func assertChecksumCopy(t *testing.T, c Config) {
	t.Helper()
	original, err := os.ReadFile(filepath.Join(c.Out, "tools", "checksums.txt"))
	if err != nil {
		t.Fatal(err)
	}
	copied, err := os.ReadFile(filepath.Join(c.Out, "checksums.txt"))
	if err != nil || string(original) != string(copied) {
		t.Fatalf("checksum manifest changed: %v", err)
	}
}

func TestGoReleaserCorruptionRejected(t *testing.T) {
	c := goreleaserFixture(t)
	testWrite(t, c.outputPath(bundleName("1.0.0", "linux-arm64")), "corrupt")
	_, err := c.BundleHashes()
	requireError(t, err)
	requireError(t, c.Assemble())
	if _, err := os.Stat(filepath.Join(c.Out, metadataName)); !os.IsNotExist(err) {
		t.Fatal("corrupt inputs produced manifest")
	}
}

func TestGoReleaserMissingAssetRejected(t *testing.T) {
	c := goreleaserFixture(t)
	if err := os.Remove(c.outputPath(executable(installerName, "1.0.0", "darwin-arm64"))); err != nil {
		t.Fatal(err)
	}
	requireError(t, c.Assemble())
}

func TestGoReleaserMalformedChecksumsRejected(t *testing.T) {
	c := goreleaserFixture(t)
	path := filepath.Join(c.Out, "bundles", "checksums.txt")
	sum := strings.Repeat("a", 64) + "  file\n"
	for _, text := range []string{"", "bad", "invalid  file", sum + sum} {
		testWrite(t, path, text)
		_, err := c.BundleHashes()
		requireError(t, err)
	}
}

func TestGoReleaserIncludesEveryRuntimeCommand(t *testing.T) {
	text := workflowSource(t, ".goreleaser-bundles.yml")
	entries, err := filepath.Glob(filepath.Join("..", "..", "cmd", "*", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		assertCommandIncluded(t, text, filepath.Base(filepath.Dir(entry)))
	}
}

func assertCommandIncluded(t *testing.T, text, name string) {
	t.Helper()
	included := strings.Contains(text, "      - "+name+"\n")
	excluded := name == installerName || name == verifierName
	if included == excluded {
		t.Fatalf("wrong archive membership for %s", name)
	}
}
