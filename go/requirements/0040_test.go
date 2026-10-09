package requirements

import "testing"

// mdpTable is a callout table that pairs with the diagram above it.
const mdpTable = "\n| # | Description | Why |\n| --- | --- | --- |\n| 6 | uses | because |\n"

// mdpDiagram wraps body in a mermaid fence under a heading.
func mdpDiagram(body string) string {
	return "# Architecture\n\n```mermaid\n" + body + "\n```\n"
}

// mdpPaired is a doc whose diagram is paired with a callout table.
func mdpPaired(body string) string {
	return mdpDiagram(body) + mdpTable
}

// mdpCheck runs `fitness-install -- mermaid-diagram-prose <args>` on
// happyRepo with files written over it.
func mdpCheck(t *testing.T, files map[string]string, args ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, append([]string{"mermaid-diagram-prose"}, args...)...)
}

// mdpError is the message a user sees for one prose label in docs/arch.md.
func mdpError(label string) string {
	return `docs/arch.md: diagram at line 3 has a relationship label with prose ("` + label + `") — put descriptions in the callout table`
}

func Test0040_1(t *testing.T) {
	t.Parallel()
	body := "Rel(app, api, \"6 Uses over HTTP\")\nBiRel(app, db, \"6 talks both ways\")\nRel_Back(api, app, '6 replies')\nRel(api, db,\n  \"6 wrapped prose\")"
	out, code := mdpCheck(t, map[string]string{"docs/arch.md": mdpPaired(body)})
	sees(t, out, code, 1, mdpError("6 Uses over HTTP"), mdpError("6 talks both ways"), mdpError("6 replies"), mdpError("6 wrapped prose"))
}

func Test0040_2(t *testing.T) {
	t.Parallel()
	out, code := mdpCheck(t, map[string]string{"docs/arch.md": mdpPaired("flowchart LR\napp -->|6 calls the api| api -->|6 reads| db")})
	sees(t, out, code, 1, mdpError("6 calls the api"), mdpError("6 reads"))
}

func Test0040_3(t *testing.T) {
	t.Parallel()
	out, code := mdpCheck(t, map[string]string{"docs/arch.md": mdpPaired("flowchart LR\napp --> api : 6 sends data   ")})
	sees(t, out, code, 1, mdpError("6 sends data"))
}

func Test0040_4(t *testing.T) {
	t.Parallel()
	body := "flowchart LR\napp -->|6| api\napi --> db : 6\ndb --> app : uses 6\n%% note : 6 something\nRel(app, api, \"6\")\nclassDef x fill:#6366f1"
	out, code := mdpCheck(t, map[string]string{"docs/arch.md": mdpPaired(body)})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0040_5(t *testing.T) {
	t.Parallel()
	out, code := mdpCheck(t, map[string]string{"docs/arch.md": mdpDiagram(`Rel(app, api, "6 Uses over HTTP")`)})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0040_6(t *testing.T) {
	t.Parallel()
	body := "Container(app, \"6 App Shell\", \"SwiftUI\")\nRel(app, api, \"6\")\na[\"6 node\"] -->|6| b"
	out, code := mdpCheck(t, map[string]string{"docs/arch.md": mdpPaired(body)})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0040_7(t *testing.T) {
	t.Parallel()
	files := map[string]string{"body.md": mdpPaired(`Rel(app, api, "6 Uses over HTTP")`)}
	out, code := mdpCheck(t, files, "--body-file", "body.md")
	sees(t, out, code, 1, `has a relationship label with prose ("6 Uses over HTTP")`)
}

func Test0040_8(t *testing.T) {
	t.Parallel()
	out, code := mdpCheck(t, nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "mermaid-diagram-prose")
}
