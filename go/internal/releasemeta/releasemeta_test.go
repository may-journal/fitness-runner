// cspell:ignore releasemeta
package releasemeta

import (
	"strings"
	"testing"
)

const fixture = ":robot: release\n---\n\n## [1.0.0](link)\n\n### Features\n\n* **release:** build verified bundles\n\n---\nfooter"

func TestFormatBodyPreservesNotes(t *testing.T) {
	body, err := FormatBody(fixture, 50)
	if err != nil {
		t.Fatal(err)
	}
	requireText(t, body, "Closes #50")
	requireText(t, body, "release: build verified bundles")
	if strings.Contains(body, "**") {
		t.Fatalf("body retains bold markup: %s", body)
	}
	notes, err := ReleaseNotes(body)
	if err != nil || !strings.Contains(notes, "### [1.0.0]") {
		t.Fatalf("notes %q: %v", notes, err)
	}
}

func requireText(t *testing.T, text, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("missing %q in %s", want, text)
	}
}

func TestFormatBodyRejectsMalformedNotes(t *testing.T) {
	if _, err := FormatBody("unexpected", 50); err == nil {
		t.Fatal("accepted malformed notes")
	}
}
