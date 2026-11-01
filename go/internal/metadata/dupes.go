package metadata

import (
	"cmp"
	_ "embed"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/aftereffect"
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/dupe"
	"github.com/may-journal/fitness-runner/go/internal/report"
)

// dupeConfig is the `dupeCheck` key of .fitnessrc.json. A zero CloseScore
// leaves the close tier off.
type dupeConfig struct {
	// WarnScore is the likeness score, from 0 to 1, at which dupe-check
	// notes that a new issue resembles an older one.
	WarnScore float64 `json:"warnScore"`
	// CloseScore is the likeness score at which dupe-check closes a new
	// issue as a duplicate of an open one; unset leaves issues open.
	CloseScore float64 `json:"closeScore"`
	// WindowDays is how many days back dupe-check looks for older issues.
	WindowDays int `json:"windowDays"`
}

// dupeFile is the part of .fitnessrc.json dupe-check reads.
type dupeFile struct {
	// DupeCheck holds dupe-check's scores and window.
	DupeCheck dupeConfig `json:"dupeCheck"`
}

// dupeSource is this file, whose doc comments describe each dupeCheck key.
//
//go:embed dupes.go
var dupeSource []byte

// dupeDefaults holds the value each dupeCheck key takes when the config
// file leaves it unset or at zero.
var dupeDefaults = dupeFile{DupeCheck: dupeConfig{WarnScore: 0.4, WindowDays: 90}}

// dupeSection is the dupeCheck keys, registered so the schema lists them.
var dupeSection = conf.Register[dupeFile](dupeDefaults, dupeSource)

// maxDupes is how many matches the note lists.
const maxDupes = 3

// loadDupeConfig reads the repo's dupeCheck key, filling in the defaults.
func loadDupeConfig(root string) (dupeConfig, error) {
	file, err := dupeSection.Load(root)
	cfg := dupeConfig{}
	if file != nil {
		cfg = file.DupeCheck
	}
	cfg.WarnScore = cmp.Or(cfg.WarnScore, dupeDefaults.DupeCheck.WarnScore)
	cfg.WindowDays = cmp.Or(cfg.WindowDays, dupeDefaults.DupeCheck.WindowDays)
	return cfg, err
}

// dupeTargets is the event's issue with the older issues it resembles. On
// workflow_dispatch it prints the score matrix of every pair instead and
// returns no targets, so a sweep writes nothing.
func dupeTargets(ev ghEvent, gh aftereffect.Client, b blobChecker) ([]target, error) {
	cfg, err := loadDupeConfig(b.root)
	if err != nil {
		return nil, err
	}
	since := time.Now().UTC().AddDate(0, 0, -cfg.WindowDays).Format(time.RFC3339)
	all, err := gh.Issues(since)
	if err != nil {
		return nil, err
	}
	pool := corpus(all, since, gh.Repo())
	if ev.name == "workflow_dispatch" {
		printMatrix(pool, cfg)
		return nil, nil
	}
	return eventDupes(ev, pool, cfg, gh.Repo())
}

// eventDupes scores the event's open issue against the older issues in the
// pool of repo.
func eventDupes(ev ghEvent, pool []entry, cfg dupeConfig, repo string) ([]target, error) {
	ts, err := eventTarget(ev.name, "issue", ev.Issue)
	if err != nil || skipDupe(ev, ts[0]) {
		return nil, err
	}
	t := ts[0]
	t.Dupes = matches(newEntry(t, pool, repo), pool, cfg.WarnScore)
	t.CloseDupe = ev.Action == "opened" && closes(t.Dupes, cfg)
	return []target{t}, nil
}

// skipDupe reports whether dupe-check leaves the event's issue alone because it
// is closed or a known bot sent the event.
func skipDupe(ev ghEvent, t target) bool {
	if t.State == "closed" || slices.Contains(KnownBots, ev.Sender.Login) {
		fmt.Printf("Skipped #%d: it is closed or a known bot changed it.\n", t.Number)
		return true
	}
	return false
}

// closes reports whether the best match is open and reaches the close score,
// for a newly opened issue only.
func closes(found []aftereffect.Dupe, cfg dupeConfig) bool {
	return cfg.CloseScore > 0 && len(found) > 0 && found[0].Open && found[0].Score >= cfg.CloseScore
}

// entry is one issue in the pool, with its words and the issues its body
// references.
type entry struct {
	target
	words dupe.Words
	refs  map[int]bool
}

// corpus is the pool of repo's issues dupe-check compares, oldest first.
func corpus(all []target, since, repo string) []entry {
	var pool []entry
	for _, t := range all {
		if inPool(t, since) {
			pool = append(pool, newEntry(t, nil, repo))
		}
	}
	return pool
}

// inPool reports whether t is open or closed since the cutoff, and neither
// opened by a known bot nor closed as a duplicate.
func inPool(t target, since string) bool {
	return (t.State == "open" || t.ClosedAt >= since) && !slices.Contains(KnownBots, t.Opener) && t.StateReason != "duplicate"
}

// newEntry is t with its words and the issues of repo it references. Its
// parent comes from its listing in pool, since an event payload never
// carries it.
func newEntry(t target, pool []entry, repo string) entry {
	for _, e := range pool {
		if e.Number == t.Number {
			t.Parent = e.Parent
		}
	}
	return entry{target: t, words: dupe.Of(t.Title, t.Body), refs: references(t.Title+"\n"+t.Body, repo)}
}

// refRe is an issue reference: #12, or a link to an issue of a repo.
var refRe = regexp.MustCompile(`(?:^|[^\w/&])#(\d+)\b|github\.com/([\w.-]+/[\w.-]+)/issues/(\d+)`)

// references are the issue numbers text names in repo.
func references(text, repo string) map[int]bool {
	refs := map[int]bool{}
	for _, m := range refRe.FindAllStringSubmatch(text, -1) {
		refs[refNumber(m, repo)] = true
	}
	return refs
}

// refNumber is the issue number a refRe match names, or 0 for a link to
// another repo.
func refNumber(m []string, repo string) int {
	if m[1] == "" && !strings.EqualFold(m[2], repo) {
		return 0
	}
	n, _ := strconv.Atoi(m[1] + m[3])
	return n
}

// relation says why a pair is never flagged, as parent, sibling or named issues,
// or is "" for unrelated issues.
func relation(a, b entry) string {
	switch {
	case nested(a, b):
		return "parent and sub-issue"
	case siblings(a, b):
		return "sub-issues of one parent"
	case a.refs[b.Number] || b.refs[a.Number]:
		return "linked"
	}
	return ""
}

// nested reports whether one of a and b is the other's parent.
func nested(a, b entry) bool {
	return a.Parent == b.Number || b.Parent == a.Number
}

// siblings reports whether a and b share a parent.
func siblings(a, b entry) bool {
	return a.Parent != 0 && a.Parent == b.Parent
}

// matches are the older, unrelated issues in pool that t scores at least
// warn against, best first, at most maxDupes.
func matches(t entry, pool []entry, warn float64) []aftereffect.Dupe {
	var found []aftereffect.Dupe
	for _, e := range pool {
		if e.Number >= t.Number || relation(t, e) != "" {
			continue
		}
		if s := dupe.Score(t.words, e.words); s >= warn {
			found = append(found, aftereffect.Dupe{ID: e.ID, Number: e.Number, Score: s, Open: e.State == "open"})
		}
	}
	slices.SortStableFunc(found, func(x, y aftereffect.Dupe) int { return cmp.Compare(y.Score, x.Score) })
	return found[:min(len(found), maxDupes)]
}

// describeDupes is the run summary line for a judged issue.
func describeDupes(t target) string {
	if len(t.Dupes) == 0 {
		return fmt.Sprintf("#%d resembles no earlier issue.", t.Number)
	}
	var parts []string
	for _, d := range t.Dupes {
		parts = append(parts, fmt.Sprintf("#%d (score %.2f)", d.Number, d.Score))
	}
	line := fmt.Sprintf("#%d may repeat %s.", t.Number, strings.Join(parts, ", "))
	if t.CloseDupe {
		line += fmt.Sprintf(" Closed it as a duplicate of #%d.", t.Dupes[0].Number)
	}
	return line
}

// pairRow is one scored pair of the sweep's matrix.
type pairRow struct {
	newer, older int
	score        float64
	tier         string
}

// printMatrix prints each pair scoring at least half the warn score, best first,
// to the log and job summary.
func printMatrix(pool []entry, cfg dupeConfig) {
	rows := scorePairs(pool, cfg)
	var b strings.Builder
	fmt.Fprintf(&b, "### dupe-check sweep\n\n%d issues; pairs scoring at least %.2f, half the warn score %.2f.\n\n", len(pool), cfg.WarnScore/2, cfg.WarnScore)
	b.WriteString("| Issue | Earlier issue | Score | Tier |\n| --- | --- | --- | --- |\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "| #%d | #%d | %.2f | %s |\n", r.newer, r.older, r.score, r.tier)
	}
	fmt.Print(b.String())
	report.WriteSummary(b.String())
}

// scorePairs scores each pair in pool, keeping those at least half the warn
// score, best first.
func scorePairs(pool []entry, cfg dupeConfig) []pairRow {
	var rows []pairRow
	for i, a := range pool {
		for _, b := range pool[:i] {
			if s := dupe.Score(a.words, b.words); s >= cfg.WarnScore/2 {
				rows = append(rows, pairRow{a.Number, b.Number, s, tier(a, b, s, cfg)})
			}
		}
	}
	slices.SortStableFunc(rows, func(x, y pairRow) int { return cmp.Compare(y.score, x.score) })
	return rows
}

// tier is what dupe-check does with a pair: skip it for a relation, close,
// warn, or nothing.
func tier(a, b entry, s float64, cfg dupeConfig) string {
	if why := relation(a, b); why != "" {
		return "skipped: " + why
	}
	switch {
	case cfg.CloseScore > 0 && s >= cfg.CloseScore:
		return "close"
	case s >= cfg.WarnScore:
		return "warn"
	}
	return "below warn"
}
