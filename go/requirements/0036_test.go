package requirements

import "testing"

// markdownLinks runs `fitness-install -- markdown-links` on happyRepo with
// files written over it.
func markdownLinks(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "markdown-links")
}

// markdownLinksReadme is the happy README with body appended; body starts on
// line 9.
func markdownLinksReadme(body string) map[string]string {
	return map[string]string{"README.md": readme + "\n" + body}
}

func Test0036_1(t *testing.T) {
	out, code := markdownLinks(t, markdownLinksReadme(`See [the guide](gone.md "Guide") and [home](README.md).`+"\n"))
	sees(t, out, code, 1, "README.md:9: broken relative link: gone.md")
}

func Test0036_2(t *testing.T) {
	files := markdownLinksReadme("See [guide](./docs/guide.md#setup), [docs](docs/), and [home](docs/../README.md).\n")
	files["docs/guide.md"] = "# Guide\n\n## Setup\n\nBack to [the readme](../README.md).\n"
	out, code := markdownLinks(t, files)
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0036_3(t *testing.T) {
	out, code := markdownLinks(t, map[string]string{"docs/guide.md": "# Guide\n\nSee [setup](setup.md#install).\n"})
	sees(t, out, code, 1, "docs/guide.md:3: broken relative link: setup.md#install")
}

func Test0036_4(t *testing.T) {
	out, code := markdownLinks(t, markdownLinksReadme("![Logo](logo.png)\n\n[spec]: spec.md\n"))
	sees(t, out, code, 1, "README.md:9: broken relative link: logo.png", "README.md:11: broken relative link: spec.md")
}

func Test0036_5(t *testing.T) {
	out, code := markdownLinks(t, markdownLinksReadme(
		"See [site](https://example.com/gone), [mail](mailto:a@example.com), [cdn](//example.com/x), and [top](#app).\n"))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0036_6(t *testing.T) {
	out, code := markdownLinks(t, markdownLinksReadme("Write `[text](gone.md)` like this.\n\n```\n[text](gone.md)\n```\n"))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0036_7(t *testing.T) {
	files := markdownLinksReadme("See [notes](docs/release%20notes.md).\n")
	files["docs/release notes.md"] = "# Release Notes\n"
	out, code := markdownLinks(t, files)
	sees(t, out, code, 0, "All 1 checks passed")
}
