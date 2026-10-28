// Command fitness-check-localized-text fails SwiftUI text and requirement
// test literals that skip the app's [String Catalog] ([#527]). The Swift
// language module owns the reading and its judgment. A literal passed to
// `Text` compiles as a [LocalizedStringKey], so a missing entry only shows in
// another language. Requirements: [0097].
//
// # Views
//
// An app opts in by tracking a `.xcstrings` file. Without one the check
// passes with zero files.
//
//   - Text views: `Text`, `Label`, `Button`, `Section`, `Link` and `ProgressView`.
//   - Input views: `TextField`, `SecureField`, `Toggle`, `Picker` and `Menu`.
//   - Container views: `LabeledContent`, `DisclosureGroup`, `GroupBox` and `ContentUnavailableView`.
//   - Title modifiers: `navigationTitle`, `navigationBarTitle`, `help` and `badge`.
//   - Other modifiers: `accessibilityLabel`, `accessibilityHint`, `alert` and `confirmationDialog`.
//   - A `verbatim:` argument counts too.
//   - A reference such as `Text(Words.Verb.share)`, a value, a labeled argument such as `systemImage: "photo"`, and a literal without letters pass.
//
// A failure reads:
//
//	App/ShareView.swift:7 - "Share" passed to Button skips the String Catalog; use a Words reference for catalog entry "verb.share"
//
// # Expressions
//
//   - A literal that is a branch of a ternary or `??` expression in a view's first argument fails.
//   - A ternary branch inside a literal's interpolation fails. Another literal there, such as a dictionary key, passes.
//   - A multi-line `"""` literal is one literal and is named whole.
//   - A test literal assigned with `=` or passed under an argument label, such as `caption: "Share"`, is plain data and passes.
//
// # Tests
//
//   - A test file sits in a folder whose name ends in `Tests`, or is named `…Tests.swift`.
//   - A test literal fails when it equals an entry's source-language words, plural and device forms included.
//   - Stale entries, entries marked not to translate, and the empty key are no labels.
//   - So test captions, prices, symbol names, and checks that a removed word is gone pass.
//
// Each failure names the file, line, and literal, the call or catalog entry,
// and the reference to use. In a pull request, or a staged commit, it judges
// only the changed Swift files against every catalog.
//
// # Reference and allow list
//
// `localizedText.reference` names the type views and tests read entries
// through, as failures name it. Without it they say `String(localized:)`.
//
// `localizedText.allow` passes a literal its `pattern` regular expression
// matches whole. Each pattern needs a `reason`, and one without, or one that
// does not compile, fails the check:
//
//	{
//	  "localizedText": {
//	    "reference": "Words",
//	    "allow": [{ "pattern": "Beta [0-9]+", "reason": "build tags stay in English for testers" }]
//	  }
//	}
//
// Org repos cannot turn the check off, as [ADR 0009] decides.
//
// [String Catalog]: https://developer.apple.com/documentation/xcode/localizing-and-varying-text-with-a-string-catalog
// [#527]: https://github.com/may-journal/fitness-runner/issues/527
// [LocalizedStringKey]: https://developer.apple.com/documentation/swiftui/localizedstringkey
// [0097]: https://github.com/may-journal/fitness-runner/blob/main/docs/requirements/0097-checks-localized-text.md
// [ADR 0009]: https://github.com/may-journal/fitness-runner/blob/main/fitness/adr-0009/README.md
package check
