// Command fitness-check-markdown-filename-camel-case requires every markdown
// file name to be camelCase, standard root docs exempt. The validation core
// is shared with the kebab-case flavor in `go/internal/mdfilename`; see
// [markdown-filename-kebab-case] for behavior and conventions covering both
// flavors.
//
// This flavor pins the camelCase convention: every `.md` basename must match
// `^[a-z][a-zA-Z0-9]*\.md$`, standard root docs and names GitHub requires
// exempt.
//
// [markdown-filename-kebab-case]: https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-markdown-filename-kebab-case/README.md
package check
