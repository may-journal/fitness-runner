// Command fitness-check-open-questions fails a list of open questions in a
// doc, page, PR or issue body. [AGENTS.md] holds the rule.
//
// # Behavior
//
//   - Fail: a heading, a bold line or a label line naming open questions, unresolved questions or questions for the owner, in any case.
//   - Fail: an `<h1>` to `<h6>` heading with that name in a tracked HTML file, or in a body.
//   - Pass: a sentence that mentions open questions, and any text in code spans, fenced blocks, HTML comments or code elements.
//   - Pass: no Markdown or HTML files, having judged 0 files.
//
// A label line, such as `Open questions:`, starts with the name and a colon
// or holds the name alone. A list item or quote counts, and so does a bold
// line that starts with the name. Each finding names its file and line. The
// [plan-structure] check shares this definition of the name.
//
// It reads every tracked Markdown and HTML file. Given `--body-file`, it
// judges that body instead, as `pr-check` and `issue-check` do; a comment
// is not judged. The Claude Code hook judges an Artifact page with it
// before the page publishes.
//
// [AGENTS.md]: https://github.com/may-journal/fitness-runner/blob/main/AGENTS.md#ask-in-chat
// [plan-structure]: https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-plan-structure/README.md
package check
