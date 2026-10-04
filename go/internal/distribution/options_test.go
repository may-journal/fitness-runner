package distribution

import (
	"flag"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestOptions(t *testing.T) {
	options, err := ParseOptions([]string{"--version", "go/v1.2.3", "--", "a b", "--write"}, "old", io.Discard)
	want := Options{Version: "go/v1.2.3", Args: []string{"a b", "--write"}}
	if err != nil || !reflect.DeepEqual(options, want) {
		t.Fatalf("options=%+v, %v", options, err)
	}
}

func TestInvalidOptions(t *testing.T) {
	for _, args := range [][]string{{"--unknown"}, {"--version"}, {"--install-only", "--", "argument"}, {"--install-only", "--install-hook"}, {"--github-action", "--install-hook"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if _, err := ParseOptions(args, testVersion, io.Discard); err == nil {
				t.Fatal("invalid option accepted")
			}
		})
	}
}

func TestHelp(t *testing.T) {
	if _, err := ParseOptions([]string{"--help"}, testVersion, io.Discard); err != flag.ErrHelp {
		t.Fatalf("help: %v", err)
	}
}

func TestVersionPins(t *testing.T) {
	for _, version := range []string{testVersion, "v" + testVersion, "go/v" + testVersion} {
		got, err := normalizeVersion(version)
		if err != nil || got != testVersion {
			t.Fatalf("version %s: %s, %v", version, got, err)
		}
	}
}

func TestRejectsMovingAndUnsafeRefs(t *testing.T) {
	for _, version := range []string{"main", "../1.2.3", "1.2", "latest", "1.2.3/extra"} {
		if _, err := normalizeVersion(version); err == nil {
			t.Fatalf("accepted %s", version)
		}
	}
}

func TestPlatforms(t *testing.T) {
	for _, system := range []string{"linux", "darwin"} {
		for _, arch := range []string{"amd64", "arm64"} {
			if _, err := Platform(system, arch); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := Platform("windows", "amd64"); err == nil {
		t.Fatal("unsupported host accepted")
	}
}

func TestManifest(t *testing.T) {
	digest := strings.Repeat("a", 64)
	for _, manifest := range []string{"", "bad  " + testAsset, digest + "  other", digest + "  " + testAsset + "\n" + digest + "  " + testAsset} {
		if _, err := manifestHash(manifest, testAsset); err == nil {
			t.Fatal("bad manifest accepted")
		}
	}
	got, err := manifestHash(digest+"  "+testAsset, testAsset)
	if err != nil || got != digest {
		t.Fatalf("hash=%s, %v", got, err)
	}
}
