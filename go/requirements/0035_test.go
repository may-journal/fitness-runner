package requirements

import (
	"strings"
	"testing"
)

// frontMatterCheck runs `fitness-install -- markdown-front-matter` on
// happyRepo with files written over it.
func frontMatterCheck(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "markdown-front-matter")
}

// frontMatterDoc is a doc whose front matter body is keys.
func frontMatterDoc(keys string) string {
	return "---\n" + keys + "\n---\n\n# Guide\n\nShort and clean.\n"
}

// frontMatterComment is front matter inside an HTML comment, as the PR
// template carries it, naming the repo's config from .github.
const frontMatterComment = "<!--\n---\nrelatedConfigurations: ['../.fitnessrc.json']\n---\n-->\n"

// frontMatterMissing is the failure a doc without either key gets.
const frontMatterMissing = "missing front matter with fitnessFunctions or relatedConfigurations"

func Test0035_1(t *testing.T) {
	out, code := frontMatterCheck(t, map[string]string{
		"plain.md":    "# Plain\n\nShort and clean.\n",
		"title.md":    frontMatterDoc("title: Notes"),
		"late.md":     "# Late\n" + frontMatterComment,
		"unclosed.md": strings.TrimSuffix(frontMatterComment, "-->\n"),
	})
	sees(t, out, code, 1, "plain.md: "+frontMatterMissing, "title.md: "+frontMatterMissing,
		"late.md: "+frontMatterMissing, "unclosed.md: "+frontMatterMissing)
}

func Test0035_2(t *testing.T) {
	out, code := frontMatterCheck(t, map[string]string{"guide.md": frontMatterDoc("fitnessFunctions: []")})
	sees(t, out, code, 1, "guide.md: fitnessFunctions must not be an empty array")
}

func Test0035_3(t *testing.T) {
	out, code := frontMatterCheck(t, map[string]string{"guide.md": frontMatterDoc("relatedConfigurations: ['./gone.json']")})
	sees(t, out, code, 1, "guide.md: front matter path missing: ./gone.json")
}

func Test0035_4(t *testing.T) {
	out, code := frontMatterCheck(t, map[string]string{"docs/guide/setup.md": frontMatterDoc("relatedConfigurations: ['../../.fitnessrc.json']")})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0035_5(t *testing.T) {
	escape := "../../../../../../../../../../etc/hosts"
	out, code := frontMatterCheck(t, map[string]string{"guide.md": frontMatterDoc("relatedConfigurations: ['" + escape + "']")})
	sees(t, out, code, 1, "guide.md: front matter path escapes repo: "+escape)
}

func Test0035_6(t *testing.T) {
	entries := "fitnessFunctions: ['markdown-front-matter', 'https://example.com/missing', '#usage', 'mailto:team@example.com']"
	out, code := frontMatterCheck(t, map[string]string{"guide.md": frontMatterDoc(entries)})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0035_7(t *testing.T) {
	out, code := frontMatterCheck(t, map[string]string{
		"node_modules/pkg/readme.md": "# Package\n",
		"dist/notes.md":              "# Notes\n",
	})
	sees(t, out, code, 1, "node_modules/pkg/readme.md: "+frontMatterMissing, "dist/notes.md: "+frontMatterMissing)
}

func Test0035_8(t *testing.T) {
	template := frontMatterComment + "\n> One clear summary.\n"
	out, code := frontMatterCheck(t, map[string]string{".github/PULL_REQUEST_TEMPLATE.md": template})
	sees(t, out, code, 0, "All 1 checks passed")
}
