// Command fitness-check-markdown-no-bold-italic fails bold and italic
// emphasis in markdown files: `**bold**`, `__bold__`, `*italic*`, or
// `_italic_`.
//
// # Behavior
//
//   - Pass: no .md files, or no markdown file contains bold (double asterisk or double underscore) or italic (single asterisk or single underscore) emphasis.
//   - Fail: any .md file contains disallowed emphasis; one error per occurrence with file path and snippet.
//
// Scans every tracked `.md` file, `CHANGELOG.md` included. Use bold or italic
// only when explicitly required. Ignores emphasis inside fenced code, inline
// code, and markdown link blocks `[text](url)`, so underscores in URLs or
// link text are not flagged. The bold scope Release Please gives release
// bullets is allowed too.
package check
