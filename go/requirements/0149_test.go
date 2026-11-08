package requirements

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Issue URLs the gh stand-in hands out for the migrated ADRs.
const (
	sqliteIssue = "https://github.com/may-journal/example/issues/41"
	syncIssue   = "https://github.com/may-journal/example/issues/42"
	filesIssue  = "https://github.com/may-journal/example/issues/43"
	oldIssue    = "https://github.com/may-journal/example/issues/44"
	cacheIssue  = "https://github.com/may-journal/example/issues/45"
)

// oldRule is a superseded check README titled with the ADR word, its folder
// holding no program.
const oldRule = "# ADR 0002: Old rule\n\nAn old rule.\n\nStatus: Superseded by [small files](../adr-0001/README.md)\n\n" +
	"## Context\n\nIt was slow.\n\n## Decision\n\nKeep it.\n\n## Consequences\n\nIt stayed slow.\n"

// migrateRepo has three ADRs in an `adr` folder, the second superseded by
// the first and the third rejected, beside the folder's template. An index
// and a guide link them, and two check READMEs hold their decisions.
func migrateRepo() map[string]string {
	return with(heldCheck(), map[string]string{
		"docs/architecture/adr/0001-use-sqlite.md": "# ADR 0001 — Use SQLite\n\nDate: 2024-05-01\n\nStatus: accepted\n\n## Context\n\nWe sync offline.\n\n" +
			"## Decision\n\nStore notes in SQLite, synced as [ADR 0002](0002-sync.md) says.\n\n## Consequences\n\nOne file holds every note.\n",
		"docs/architecture/adr/0002-sync.md": "# ADR 0002 — Sync whole files\n\n> Whole files sync simply.\n\n" +
			"* Status: superseded by [ADR 0001](/docs/architecture/adr/0001-use-sqlite.md)\n\n" +
			"## context\n\nNotes change offline.\n\n## decision\n\nSync whole files.\n\n## consequences\n\nTwo edits clash.\n",
		"docs/architecture/adr/0003-cache.md": "# ADR 0003 — Cache notes\n\nStatus: rejected\n\n## Context\n\nReads are slow.\n\n" +
			"## Decision\n\nCache every note.\n\n## Consequences\n\nThe cache goes stale.\n",
		"docs/architecture/adr/template.md": "# ADR NNNN — Title\n\nStatus: proposed\n\n## Context\n\n## Decision\n\n## Consequences\n",
		"docs/architecture/adr/README.md":   "# ADRs\n\n- [Use SQLite](0001-use-sqlite.md)\n- [Sync](0002-sync.md)\n",
		"docs/guide.md":                     "# Guide\n\nRead [the storage decision](architecture/adr/0001-use-sqlite.md#decision) and [small files](../fitness/adr-0001/).\n",
		"fitness/adr-0002/README.md":        oldRule,
	})
}

// migrateGitHub is the saved GitHub for a migration: the ADR issues listed,
// and new ones opened in turn.
func migrateGitHub(listed string) []response {
	return []response{
		{Match: []string{"repo view"}, Stdout: "may-journal/example\n"},
		{Match: []string{"issue list", "--label ADR"}, Stdout: listed},
		{Match: []string{"label create ADR"}},
		{Match: []string{"issue create", "[ADR] Use SQLite"}, Stdout: sqliteIssue + "\n"},
		{Match: []string{"issue create", "[ADR] Sync whole files"}, Stdout: syncIssue + "\n"},
		{Match: []string{"issue create", "[ADR] Small files"}, Stdout: filesIssue + "\n"},
		{Match: []string{"issue create", "[ADR] Old rule"}, Stdout: oldIssue + "\n"},
		{Match: []string{"issue create", "[ADR] Cache notes"}, Stdout: cacheIssue + "\n"},
		{Match: []string{"issue edit"}},
		{Match: []string{"issue close"}},
	}
}

// listedIssues is the `gh issue list` output for ADR issues, each URL with
// the file its marker names, the body before it, and its state.
func listedIssues(t *testing.T, issues ...[4]string) string {
	t.Helper()
	list := []map[string]string{}
	for _, i := range issues {
		list = append(list, map[string]string{"url": i[0], "body": i[2] + "\n\n<!-- fitness:adr-migrate:" + i[1] + " -->", "state": i[3]})
	}
	out, err := json.Marshal(list)
	mustDo(t, err)
	return string(out) + "\n"
}

// keptSQLite is the listed issue an earlier migration opened for the first
// ADR, still linking the second ADR's file on the branch.
func keptSQLite(t *testing.T) [4]string {
	t.Helper()
	return [4]string{sqliteIssue, "docs/architecture/adr/0001-use-sqlite.md",
		"Synced as [ADR 0002](https://github.com/may-journal/example/blob/main/docs/architecture/adr/0002-sync.md#decision).", "OPEN"}
}

// migrate runs `fitness adr-migrate` with args in repo against saved.
func migrate(t *testing.T, repo string, saved []response, args ...string) (string, int, replays) {
	t.Helper()
	rp := standIn(t, map[string][]response{"gh": saved})
	out, code := fitness(t, repo, rp.env, append([]string{"adr-migrate"}, args...)...)
	return out, code, rp
}

func Test0149_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", migrateRepo())
	out, code, rp := migrate(t, repo, migrateGitHub(listedIssues(t, keptSQLite(t))), "--dry-run")
	sees(t, out, code, 0,
		"kept docs/architecture/adr/0001-use-sqlite.md: "+sqliteIssue, "would edit "+sqliteIssue,
		`would open "[ADR] Small files" from fitness/adr-0001/README.md`,
		"would remove docs/architecture/adr/0002-sync.md",
		"would close the issue for docs/architecture/adr/0002-sync.md as superseded",
		"would close the issue for docs/architecture/adr/0003-cache.md as rejected",
		"would rewrite docs/guide.md", "would rewrite fitness/adr-0001/README.md")
	if calls := rp.calls("gh"); len(calls) != rp.count("gh", "repo view")+rp.count("gh", "issue list") {
		t.Errorf("a dry run must only read GitHub, got gh calls %q", calls)
	}
	if status, err := exec.Command("git", "-C", repo, "status", "--porcelain").Output(); err != nil || len(status) > 0 {
		t.Errorf("a dry run must change no file, got:\n%s", status)
	}
}

func Test0149_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", migrateRepo())
	out, code, rp := migrate(t, repo, migrateGitHub(listedIssues(t)))
	sees(t, out, code, 0, "opened "+sqliteIssue+" from docs/architecture/adr/0001-use-sqlite.md",
		"removed docs/architecture/adr/0001-use-sqlite.md", "rewrote docs/guide.md")
	wantCall(t, rp, "an ADR issue opened with the file's sections", "issue create", "--label ADR", "[ADR] Use SQLite",
		"> Use SQLite\\n\\n## Background\\n\\nWe sync offline.", "## Decision\\n\\nStore notes in SQLite",
		"## Currently Known Consequences\\n\\nOne file holds every note.")
	if rp.called("gh", "issue create", "## Status") {
		t.Errorf("want no Status section, as the issue's state is its status, got gh calls %q", rp.calls("gh"))
	}
	wantCall(t, rp, "the superseded ADR's issue opened with its quoted summary as the pitch", "issue create", "[ADR] Sync whole files",
		"--body-file - > Whole files sync simply.\\n\\n## Background\\n\\nNotes change offline.")
	wantCall(t, rp, "the rejected ADR's issue closed", "issue close "+cacheIssue, "--reason not planned", "Rejected, as the file it came from said.")
	wantCall(t, rp, "the superseded ADR's issue closed naming the issue that replaces it",
		"issue close "+syncIssue, "--reason not planned", "Superseded by "+sqliteIssue)
	wantCall(t, rp, "the superseded check's issue closed naming the issue that replaces it",
		"issue close "+oldIssue, "--reason not planned", "Superseded by "+filesIssue)
	if rp.called("gh", "issue close "+sqliteIssue) {
		t.Errorf("want an accepted ADR's issue left open, got gh calls %q", rp.calls("gh"))
	}
	wantCall(t, rp, "the first issue's link to the second ADR pointed at its issue", "issue edit "+sqliteIssue, "as [ADR 0002]("+syncIssue+") says")
	wantFiles(t, repo, map[string]string{
		"docs/guide.md":                     "# Guide\n\nRead [the storage decision](" + sqliteIssue + ") and [small files](" + filesIssue + ").\n",
		"docs/architecture/adr/README.md":   "# ADRs\n\n- [Use SQLite](" + sqliteIssue + ")\n- [Sync](" + syncIssue + ")\n",
		"fitness/adr-0001/README.md":        "# 0001 — Small files\n\nSmall files read fast.\n\n" + filesIssue + "\n\n## Check\n\nFails a long Markdown file.\n",
		"fitness/adr-0002/README.md":        "# Old rule\n\nAn old rule.\n\n" + oldIssue + "\n\nStatus: Superseded by " + filesIssue + "\n",
		"docs/architecture/adr/template.md": migrateRepo()["docs/architecture/adr/template.md"],
	})
	if _, err := os.Stat(filepath.Join(repo, "docs", "architecture", "adr", "0002-sync.md")); !os.IsNotExist(err) {
		t.Errorf("want the migrated ADR file removed, got %v", err)
	}
	out, code = fitness(t, repo, nil, "adr-as-issue")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0149_3(t *testing.T) {
	t.Parallel()
	listed := listedIssues(t, keptSQLite(t), [4]string{syncIssue, "docs/architecture/adr/0002-sync.md", "Sync whole files.", "CLOSED"})
	repo := example(t, "happyRepo", with(migrateRepo(), map[string]string{
		".github/ISSUE_TEMPLATE/adr.md": "---\nname: ADR\nabout: A decision.\nlabels: ADR\n---\n\n> REPLACE-ME\n\n## Why\n\n## Decision\n\n## Effects\n",
	}))
	out, code, rp := migrate(t, filepath.Join(repo, "docs"), migrateGitHub(listed), "--repo=may-journal/example")
	sees(t, out, code, 0, "kept docs/architecture/adr/0001-use-sqlite.md: "+sqliteIssue)
	wantCall(t, rp, "a new issue under the repo template's headings", "issue create", "[ADR] Small files",
		"## Why\\n\\n", "## Decision\\n\\n", "## Effects\\n\\n")
	if rp.called("gh", "issue create", "[ADR] Use SQLite") || rp.called("gh", "issue create", "[ADR] Sync whole files") {
		t.Errorf("want the existing issues kept, not opened again, got gh calls %q", rp.calls("gh"))
	}
	wantCall(t, rp, "the kept issue's link to a moved ADR file pointed at its issue", "issue edit "+sqliteIssue, "Synced as [ADR 0002]("+syncIssue+").")
	if rp.called("gh", "issue close "+syncIssue) {
		t.Errorf("want a closed superseded issue left as it is, got gh calls %q", rp.calls("gh"))
	}
	out, code, rp = migrate(t, repo, migrateGitHub(listed), "--repo=may-journal/example")
	sees(t, out, code, 0, "no ADR files to move")
	if rp.called("gh", "issue create") {
		t.Errorf("a rerun after the files moved must open nothing, got gh calls %q", rp.calls("gh"))
	}
}

// wantCall asserts some gh call holds every want, naming what it shows.
func wantCall(t *testing.T, rp replays, what string, want ...string) {
	t.Helper()
	if !rp.called("gh", want...) {
		t.Errorf("want %s, got gh calls %q", what, rp.calls("gh"))
	}
}

// wantFiles asserts each file in repo holds its text.
func wantFiles(t *testing.T, repo string, want map[string]string) {
	t.Helper()
	for name, text := range want {
		got, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(name)))
		if err != nil || string(got) != text {
			t.Errorf("%s = %q (%v), want %q", name, got, err, text)
		}
	}
}
