// Command fitness-check-doc-template makes each markdown document match the
// shape of a template that governs it. A template file defines a markdown
// document's shape, and its neighbors must match it. A folder may hold
// several templates, one per kind of doc. This is the file-level counterpart
// of `plan-structure` and `pr-structure`, sharing the `mdtemplate` engine.
//
// # What is a template
//
//   - `template.md` or `*.template.md` — a template that governs its neighbors.
//   - Any `*.md` directly in a `templates` subfolder — a template for that subfolder's parent folder.
//   - A GitHub PR or Issue template under `.github` (`PULL_REQUEST_TEMPLATE.md`, `ISSUE_TEMPLATE/*.md`).
//
// # Rules
//
//   - Every `##` section of a template must carry a guiding `<!-- comment -->`.
//   - A folder's templates are its own `template.md`-style files plus every file in its `templates` subfolder.
//   - Walking up from each file's own folder, the first folder with a template that governs it decides.
//   - The file must have the same `##` sections as any one of that folder's templates.
//   - So `docs/arch/adr/0001-x.md` follows `docs/arch/adr/template.md`, while `docs/arch/04-code.md` follows `docs/arch/templates/*.md`.
//   - A GitHub template gets the comment rule only; its body is checked by `pr-structure` and `plan-structure`.
//
// # Named templates
//
// A file in a `templates` folder governs alone the neighbor that shares its
// name, so `docs/architecture/templates/04-code.md` judges
// `docs/architecture/04-code.md`. It governs no other doc, so the other
// neighbors match the folder's other templates. A neighbor in a folder of
// only named templates follows the nearest folder above with others, so
// `docs/architecture/notes.md` follows `docs/template.md`. A named template
// holds its namesake to its whole shape:
//
//   - Sections: the template's `##` sections open the doc, in order. Sections of the doc's own may follow.
//   - Legends: a section with a legend in the template needs a legend of the same Mermaid type.
//   - Legend kinds: a diagram with no callout numbers, one titled `Legend`, or one under a `Legend` heading.
//   - Callout tables: a section with one in the template needs one, and each has the template's columns.
//   - Captions: the paragraph above each callout table starts with the first sentence of the template's caption.
//
// The five architecture view templates ship this way; the
// [architecture-contract README] says how to copy them.
//
// # Behavior
//
//   - Pass: every template guides each section, and every governed file matches. A repo with no templates passes.
//   - Fail: a template section lacks a comment, or a governed file matches none of its templates.
//   - With one template, each difference is its own error. With several, one error names them all.
//
// [architecture-contract README]: https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-architecture-contract/README.md#templates
package check
