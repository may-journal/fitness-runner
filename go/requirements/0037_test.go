package requirements

import "testing"

// noBoldItalic runs `fitness-install -- markdown-no-bold-italic <args>` on
// happyRepo with files written over it.
func noBoldItalic(t *testing.T, files map[string]string, args ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, append([]string{"markdown-no-bold-italic"}, args...)...)
}

// noBoldItalicReadme runs the check with text added to happyRepo's README.
func noBoldItalicReadme(t *testing.T, text string) (string, int) {
	t.Helper()
	return noBoldItalic(t, map[string]string{"README.md": readme + "\n" + text})
}

// noBoldItalicRule is the advice every finding carries.
const noBoldItalicRule = " (use only when explicitly required): "

func Test0037_1(t *testing.T) {
	out, code := noBoldItalicReadme(t, "Some **strong** and *soft* words.\n")
	sees(t, out, code, 1,
		`README.md: disallowed **bold**`+noBoldItalicRule+`"**strong**"`,
		`README.md: disallowed *italic*`+noBoldItalicRule+`"*soft*"`)
}

func Test0037_2(t *testing.T) {
	out, code := noBoldItalicReadme(t, "Some __strong__ and _soft_ words.\n")
	sees(t, out, code, 1,
		`README.md: disallowed __bold__`+noBoldItalicRule+`"__strong__"`,
		`README.md: disallowed _italic_`+noBoldItalicRule+`"_soft_"`)
}

func Test0037_3(t *testing.T) {
	out, code := noBoldItalicReadme(t, "Use `**strong**` here.\n\n```\n_soft_ and **strong**\n```\n")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0037_4(t *testing.T) {
	out, code := noBoldItalicReadme(t, "See [the _dev_ path](https://example.com/_dev_/to_page_).\n")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0037_5(t *testing.T) {
	out, code := noBoldItalicReadme(t, "* first item\n* second item\n* third item\n")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0037_6(t *testing.T) {
	changelog := "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# Changelog\n\n## Changes\n\n### 2026.10.08.1400\n\n- Docs: a **loud** note.\n"
	out, code := noBoldItalic(t, map[string]string{"CHANGELOG.md": changelog})
	sees(t, out, code, 1, `CHANGELOG.md: disallowed **bold**`+noBoldItalicRule+`"**loud**"`)
}

func Test0037_7(t *testing.T) {
	out, code := noBoldItalic(t, map[string]string{"body.md": "Hello _there_.\n"}, "--body-file", "body.md")
	sees(t, out, code, 1, `(description): disallowed _italic_`+noBoldItalicRule+`"_there_"`)
}
