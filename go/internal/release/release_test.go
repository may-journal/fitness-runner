package release

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionAndTag(t *testing.T) {
	c := testConfig(t)
	version, err := c.Version()
	if err != nil || version != "1.0.0" {
		t.Fatalf("version %s: %v", version, err)
	}
	if err := c.VerifyTag("v1.0.0"); err != nil {
		t.Fatal(err)
	}
	requireError(t, c.VerifyTag("go/v1.0.0"))
	requireError(t, c.VerifyTag("v1.0.1"))
}

func TestRejectMalformedVersionFile(t *testing.T) {
	c := testConfig(t)
	for _, version := range []string{"", "v1.0.0", "1.2", "1.2.03", "1.0.0;command"} {
		testWrite(t, filepath.Join(c.Root, "version.txt"), version)
		_, err := c.Version()
		requireError(t, err)
	}
}

func TestAssetHashValidation(t *testing.T) {
	metadata := Metadata{Assets: []Asset{{Name: "installer", Hash: strings.Repeat("0", 64)}}}
	for _, name := range []string{"missing", "installer"} {
		if err := verifyAsset(metadata, name, []byte("corrupt")); err == nil {
			t.Fatal("bad asset accepted")
		}
	}
}
