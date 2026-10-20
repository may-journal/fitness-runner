package requirements

import (
	"maps"
	"testing"
)

// localizedCatalog is the fixture app's String Catalog: a live verb, a
// noun with plural forms, a stale word the code no longer uses, and the
// empty key Xcode keeps.
const localizedCatalog = `{
  "sourceLanguage" : "en",
  "strings" : {
    "" : { "shouldTranslate" : false },
    "noun.photo" : {
      "localizations" : { "en" : { "variations" : { "plural" : {
        "one" : { "stringUnit" : { "state" : "translated", "value" : "Photo" } },
        "other" : { "stringUnit" : { "state" : "translated", "value" : "Photos" } }
      } } } }
    },
    "verb.archive" : {
      "extractionState" : "stale",
      "localizations" : { "en" : { "stringUnit" : { "state" : "translated", "value" : "Archive" } } }
    },
    "verb.share" : {
      "extractionState" : "extracted_with_value",
      "localizations" : { "en" : { "stringUnit" : { "state" : "new", "value" : "Share" } } }
    }
  },
  "version" : "1.0"
}
`

// localizedWords is the fixture app's reference type, which reads each
// label from the catalog by key.
const localizedWords = `import Foundation

enum Words {
    enum Verb {
        static var share: String {
            String(localized: "verb.share", defaultValue: "Share", comment: "Verb: share a photo")
        }
    }

    enum Noun {
        static var photo: String {
            String(localized: "noun.photo", defaultValue: "Photo", comment: "Noun: one photo")
        }
    }
}
`

// localizedApp is the fixture app with a String Catalog, its Words
// reference, and a config naming that reference, with files written over
// it. An empty body deletes a fixture file.
func localizedApp(t *testing.T, files map[string]string) string {
	t.Helper()
	app := map[string]string{
		".fitnessrc.json":           `{"localizedText": {"reference": "Words"}}` + "\n",
		"App/Localizable.xcstrings": localizedCatalog,
		"App/Words.swift":           localizedWords,
	}
	maps.Copy(app, files)
	return example(t, "happyRepo", app)
}

// localizedText runs `fitness-install -- localized-text` on the fixture
// app with files written over it.
func localizedText(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, localizedApp(t, files), nil, "localized-text")
}

// localizedView is a view that passes literals to a button and a title.
const localizedView = `import SwiftUI

struct ShareView: View {
    let count: Int

    var body: some View {
        Button("Share") {}
            .navigationTitle("Library")
        Text("\(count) shared")
    }
}
`

// localizedStateView passes literals as ternary and ?? branches.
const localizedStateView = `import SwiftUI

struct StateView: View {
    let isShared: Bool
    let name: String?

    var body: some View {
        Text(isShared ? "Shared" : "Private")
        Text(name ?? "Nobody")
        Text(isShared ? Words.Verb.share : Words.Noun.photo)
        Label("Photos", systemImage: isShared ? "person" : "lock")
    }
}
`

// localizedCountView passes literals as branches inside an interpolation.
const localizedCountView = `import SwiftUI

struct CountView: View {
    let count: Int

    var body: some View {
        Text("\(count == 1 ? "photo" : "photos")")
        Text("\(count)\(Words.Noun.photo)")
    }
}
`

func Test0097_1(t *testing.T) {
	t.Parallel()
	out, code := localizedText(t, map[string]string{
		"App/ShareView.swift": localizedView,
		"App/StateView.swift": localizedStateView,
		"App/CountView.swift": localizedCountView,
		"App/HelpView.swift":  "import SwiftUI\n\nstruct HelpView: View {\n    var body: some View {\n        Text(\"\"\"\n            Share a \"photo\"\n            with a friend\n            \"\"\")\n    }\n}\n",
	})
	sees(t, out, code, 1,
		`App/StateView.swift:8 - "Shared" passed to Text skips the String Catalog; add it to the String Catalog and use a Words reference`,
		`App/StateView.swift:8 - "Private" passed to Text skips the String Catalog; add it to the String Catalog and use a Words reference`,
		`App/StateView.swift:9 - "Nobody" passed to Text skips the String Catalog; add it to the String Catalog and use a Words reference`,
		`App/StateView.swift:11 - "Photos" passed to Label skips the String Catalog; use a Words reference for catalog entry "noun.photo"`,
		`App/CountView.swift:7 - "photo" passed to Text skips the String Catalog; add it to the String Catalog and use a Words reference`,
		`App/CountView.swift:7 - "photos" passed to Text skips the String Catalog; add it to the String Catalog and use a Words reference`,
		"App/HelpView.swift:5 - \"\"\"\n            Share a \"photo\"\n            with a friend\n            \"\"\" passed to Text skips the String Catalog; add it to the String Catalog and use a Words reference",
		`App/ShareView.swift:7 - "Share" passed to Button skips the String Catalog; use a Words reference for catalog entry "verb.share"`,
		`App/ShareView.swift:8 - "Library" passed to navigationTitle skips the String Catalog; add it to the String Catalog and use a Words reference`,
		`App/ShareView.swift:9 - "\(count) shared" passed to Text skips the String Catalog; add it to the String Catalog and use a Words reference`)
}

func Test0097_2(t *testing.T) {
	t.Parallel()
	out, code := localizedText(t, map[string]string{"App/PhotoView.swift": `import SwiftUI

struct PhotoView: View {
    let caption: String
    let count: Int
    let titles: [String: String]

    var body: some View {
        VStack {
            Label(Words.Noun.photo, systemImage: "photo")
            Text(caption)
            Text("·")
            Text("\(count)")
            Text("\(titles["first"] ?? Words.Noun.photo)")
            Button(Words.Verb.share) {}
                .accessibilityLabel(Words.Verb.share)
        }
    }
}
`})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0097_3(t *testing.T) {
	t.Parallel()
	out, code := localizedText(t, map[string]string{"Tests/Requirements/ShareTests.swift": `import Testing

struct ShareTests {
    func tapsShare(_ user: User) async throws {
        try await user.taps(.button("Share"))
    }
}
`})
	sees(t, out, code, 1,
		`Tests/Requirements/ShareTests.swift:5 - "Share" types catalog entry "verb.share"; read it through a Words reference`)
}

func Test0097_4(t *testing.T) {
	t.Parallel()
	out, code := localizedText(t, map[string]string{"Tests/Requirements/Support/Guest.swift": `struct Guest {
    let user: User

    func sharesCaptionedPhoto() async throws {
        try await user.writes("Swim at the lake")
        try await user.sees(.text("$9.99"))
        try await user.sees(.image("Sparkle"))
        try await user.doesNotSee(.button("Archive"))
        try await user.taps(.button(Words.Verb.share))
        let label = "Share"
        try await user.writes(label)
        try await user.adds(Photo(title: "Photos", caption: "Share"))
    }
}
`})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0097_5(t *testing.T) {
	t.Parallel()
	out, code := localizedText(t, map[string]string{
		".fitnessrc.json": `{"localizedText": {"reference": "Words", "allow": [
  {"pattern": "Beta [0-9]+", "reason": "build tags stay in English for testers"}
]}}` + "\n",
		"App/BuildTag.swift": "import SwiftUI\n\nstruct BuildTag: View {\n    var body: some View { Text(\"Beta 3\") }\n}\n",
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0097_6(t *testing.T) {
	t.Parallel()
	out, code := localizedText(t, map[string]string{
		".fitnessrc.json": `{"localizedText": {"reference": "Words", "allow": [
  {"pattern": "Beta [0-9]+"},
  {"pattern": "(Beta", "reason": "build tags stay in English for testers"}
]}}` + "\n",
	})
	sees(t, out, code, 1,
		`localizedText.allow pattern "Beta [0-9]+" needs a reason`,
		`localizedText.allow pattern "(Beta" is not a regular expression`)
}

func Test0097_7(t *testing.T) {
	t.Parallel()
	out, code := localizedText(t, map[string]string{
		"App/Localizable.xcstrings": "",
		"App/ShareView.swift":       localizedView,
	})
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "localized-text")
}
