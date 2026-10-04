package distribution

import "testing"

func TestReleaseTagCompatibility(t *testing.T) {
	cases := map[string]string{
		"0.20261004.1140": "go/v0.20261004.1140",
		"1.0.0":           "v1.0.0",
		"2.10.3":          "v2.10.3",
		"0.1.0":           "v0.1.0",
	}
	for version, tag := range cases {
		if got := ReleaseTag(version); got != tag {
			t.Fatalf("tag for %s: %s", version, got)
		}
		installer := Installer{ReleaseURL: "https://example.com/releases"}
		if got := installer.assetURL(version, "checksums.txt"); got != "https://example.com/releases/download/"+tag+"/checksums.txt" {
			t.Fatalf("download route for %s: %s", version, got)
		}
	}
}
