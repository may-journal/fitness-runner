package requirements

import (
	"cmp"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// perfCheck is one check's entry in the `--json` report, as a reader of the
// file sees it.
type perfCheck struct {
	Name         string `json:"name"`
	Ok           bool   `json:"ok"`
	Ms           int64  `json:"ms"`
	TimeoutMs    int64  `json:"timeoutMs"`
	TimedOut     bool   `json:"timedOut"`
	FilesChecked int    `json:"filesChecked"`
	BinaryBytes  int64  `json:"binaryBytes"`
	Errors       int    `json:"errors"`
}

// perfReport is the `--json` report file.
type perfReport struct {
	ElapsedMs   int64       `json:"elapsedMs"`
	RunnerBytes int64       `json:"runnerBytes"`
	Checks      []perfCheck `json:"checks"`
}

// withReport runs the full suite with `--json` in repo and returns what it
// printed and the report it wrote.
func withReport(t *testing.T, repo string) (string, perfReport) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "perf", "fitness.json")
	out, _ := fitness(t, repo, nil, "--policy=external", "--all", "--json="+path)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("fitness --json wrote no report: %v\n%s", err, out)
	}
	var report perfReport
	mustDo(t, json.Unmarshal(raw, &report))
	return out, report
}

// reported returns the report's entry for check.
func reported(t *testing.T, report perfReport, check string) perfCheck {
	t.Helper()
	for _, c := range report.Checks {
		if c.Name == check {
			return c
		}
	}
	t.Fatalf("the report has no entry for %s", check)
	return perfCheck{}
}

func Test0077_1(t *testing.T) {
	t.Parallel()
	out, report := withReport(t, example(t, "happyRepo", misspelled(1)))
	if report.RunnerBytes <= 0 || report.ElapsedMs <= 0 {
		t.Errorf("the report must give the runner's size and the run's time, got %+v", report)
	}
	for _, c := range report.Checks {
		fullEntry(t, out, c)
	}
	verdicts(t, report)
}

// verdicts asserts the misspelling failed cspell while prose-budget passed.
func verdicts(t *testing.T, report perfReport) {
	t.Helper()
	if cspell := reported(t, report, "cspell"); verdictOf(cspell) != (verdict{ok: false, judged: true, errors: 1}) {
		t.Errorf("cspell must fail with one error having judged files, got %+v", cspell)
	}
	if prose := reported(t, report, "prose-budget"); verdictOf(prose) != (verdict{ok: true, judged: prose.FilesChecked > 0, errors: 0}) {
		t.Errorf("prose-budget must pass with no errors, got %+v", prose)
	}
}

// verdict is what a reader takes from one report entry: the result, whether
// files were judged, and how many errors it found.
type verdict struct {
	ok, judged bool
	errors     int
}

func verdictOf(c perfCheck) verdict {
	return verdict{ok: c.Ok, judged: c.FilesChecked > 0, errors: c.Errors}
}

// fullEntry asserts a report entry matches a table row and carries its
// budget, binary size, and time.
func fullEntry(t *testing.T, out string, c perfCheck) {
	t.Helper()
	if row(out, c.Name) == "" {
		t.Errorf("the report lists %s, which the table never showed", c.Name)
	}
	if c.TimeoutMs <= 0 || c.BinaryBytes <= 0 || c.Ms < 0 {
		t.Errorf("%s must carry its budget, binary size, and time, got %+v", c.Name, c)
	}
}

func Test0077_2(t *testing.T) {
	t.Parallel()
	_, report := withReport(t, example(t, "happyRepo", hungTool("300", "")))
	prettier := reported(t, report, "prettier")
	if !prettier.TimedOut || prettier.Ok || prettier.TimeoutMs != 300 {
		t.Errorf("prettier must be timed out at its 300ms budget, got %+v", prettier)
	}
	if goTest := reported(t, report, "go-test"); goTest.TimedOut {
		t.Errorf("only the hung check times out, got %+v", goTest)
	}
}

func Test0077_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	out, code := fitness(t, repo, nil, "--policy=external", "--json")
	sees(t, out, code, 1, "use --json=<file> once")
	blocked := filepath.Join(t.TempDir(), "file")
	mustDo(t, os.WriteFile(blocked, []byte("x"), 0o644))
	out, code = fitness(t, repo, nil, "--policy=external", "--all", "--json="+filepath.Join(blocked, "fitness.json"))
	sees(t, out, code, 1, "fitness --json")
}

// timingRow matches one line of the job summary's timing table.
var timingRow = regexp.MustCompile(`^\| (\S+) \| (\d+)ms \| (\d+)ms \| \d+% \|$`)

func Test0077_4(t *testing.T) {
	t.Parallel()
	out, _, summary := inActions(t, example(t, "happyRepo", nil))
	sees(t, summary, 0, 0, "### Check timing", "| Check | Time | Budget | Used |")
	times := timings(summary)
	if want := strings.Count(out, "  → "); len(times) != want {
		t.Errorf("timing table has %d checks, the run started %d:\n%s", len(times), want, summary)
	}
	if !slices.IsSortedFunc(times, func(a, b int64) int { return cmp.Compare(b, a) }) {
		t.Errorf("timing must run slowest first, got %v", times)
	}
}

// timings returns the times in the summary's timing table, in order.
func timings(summary string) []int64 {
	var times []int64
	for _, line := range strings.Split(summary, "\n") {
		if m := timingRow.FindStringSubmatch(line); m != nil {
			ms, _ := strconv.ParseInt(m[2], 10, 64)
			times = append(times, ms)
		}
	}
	return times
}

// diagramBox matches one check's box in the page: its result class, then its
// tooltip naming the check, its value, and its result.
var diagramBox = regexp.MustCompile(`<li class="([a-z-]*)"[^>]*><div class="cell" tabindex="0" title="([^:"]+): ([^,"]+), ([a-z ]+)">`)

// outsideRequest matches anything that would make the page run code or load
// another file.
var outsideRequest = regexp.MustCompile(`<script|http|src=`)

// diagramViews are the page's three ways to size the boxes.
var diagramViews = []string{"time", "size", "errors"}

func Test0077_6(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", misspelled(2))
	_, report := withReport(t, repo)
	page := drawn(t, repo, report)
	if outsideRequest.MatchString(page) {
		t.Error("the page must be self-contained, with no scripts or outside requests")
	}
	views := strings.Split(page, `<section class="view view-`)[1:]
	if len(views) != len(diagramViews) {
		t.Fatalf("the page must offer %v views, got %d", diagramViews, len(views))
	}
	for i, view := range views {
		oneBoxEach(t, diagramViews[i], view, report)
	}
	if !strings.Contains(views[2], `title="cspell: 2 errors, failed"`) {
		t.Error("the errors view must size cspell by its 2 errors and mark it failed")
	}
}

// drawn writes the report to a file, runs `fitness boxes` on it the way a
// user does, and returns the page.
func drawn(t *testing.T, repo string, report perfReport) string {
	t.Helper()
	dir := t.TempDir()
	in, out := filepath.Join(dir, "fitness.json"), filepath.Join(dir, "page", "fitness.html")
	raw, err := json.Marshal(report)
	mustDo(t, err)
	mustDo(t, os.WriteFile(in, raw, 0o644))
	said, code := fitness(t, repo, nil, "boxes", in, out)
	sees(t, said, code, 0)
	page, err := os.ReadFile(out)
	mustDo(t, err)
	return string(page)
}

// oneBoxEach asserts a view draws every reported check once, with its result
// in words and a matching class.
func oneBoxEach(t *testing.T, name, view string, report perfReport) {
	t.Helper()
	seen := map[string]int{}
	for _, m := range diagramBox.FindAllStringSubmatch(view, -1) {
		seen[m[2]]++
		saysResult(t, name, m[1], m[4], reported(t, report, m[2]))
	}
	for _, c := range report.Checks {
		if seen[c.Name] != 1 {
			t.Errorf("%s view draws %s %d times, want once", name, c.Name, seen[c.Name])
		}
	}
}

// saysResult asserts a box names its check's result in words, and carries a
// failure class exactly when the check did not pass.
func saysResult(t *testing.T, view, class, words string, c perfCheck) {
	t.Helper()
	if got, want := [2]any{words, class == ""}, [2]any{resultWords(c), c.Ok}; got != want {
		t.Errorf("%s view: %s box says %q with class %q, want %q", view, c.Name, words, class, want[0])
	}
}

// resultWords is how the page names a check's result.
func resultWords(c perfCheck) string {
	switch {
	case c.TimedOut:
		return "timed out"
	case c.Ok:
		return "passed"
	}
	return "failed"
}

func Test0077_7(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	dir := t.TempDir()
	page := filepath.Join(dir, "fitness.html")
	out, code := fitness(t, repo, nil, "boxes", filepath.Join(dir, "missing.json"), page)
	sees(t, out, code, 1, "fitness boxes", "missing.json")
	notReport := filepath.Join(dir, "other.json")
	mustDo(t, os.WriteFile(notReport, []byte(`{"name": "app"}`), 0o644))
	out, code = fitness(t, repo, nil, "boxes", notReport, page)
	sees(t, out, code, 1, "not a fitness --json report")
	out, code = fitness(t, repo, nil, "boxes", notReport)
	sees(t, out, code, 1, "use fitness boxes <report.json> <page.html>")
	if _, err := os.Stat(page); err == nil {
		t.Error("a failed run must not write a page")
	}
}

// releaseRecord is what `fitness-release record` prints.
type releaseRecord struct {
	Version   string `json:"version"`
	Platform  string `json:"platform"`
	InstallMs int64  `json:"installMs"`
	Assets    []struct {
		Name  string `json:"name"`
		Bytes int64  `json:"bytes"`
	} `json:"assets"`
}

func Test0077_5(t *testing.T) {
	t.Parallel()
	built := filepath.Dir(installer(t))
	repo := t.TempDir()
	mustDo(t, os.Symlink(built, filepath.Join(repo, "out")))
	record := recorded(t, repo)
	describesHost(t, record)
	if len(record.Assets) == 0 {
		t.Fatal("record lists no assets")
	}
	for _, a := range record.Assets {
		realSize(t, filepath.Join(built, a.Name), a.Bytes)
	}
}

// recorded runs `fitness-release record` in repo and decodes what it printed.
func recorded(t *testing.T, repo string) releaseRecord {
	t.Helper()
	out, code := releaseToolRun(t, repo, nil, "record")
	var record releaseRecord
	start, end := strings.Index(out, "{"), strings.LastIndex(out, "}")
	if code != 0 || start < 0 || json.Unmarshal([]byte(out[start:end+1]), &record) != nil {
		t.Fatalf("record must exit 0 printing JSON, got exit %d:\n%s", code, out)
	}
	return record
}

// describesHost asserts the record names this platform, the version, and an
// install time.
func describesHost(t *testing.T, record releaseRecord) {
	t.Helper()
	if record.Platform != runtime.GOOS+"-"+runtime.GOARCH || record.InstallMs <= 0 || record.Version == "" {
		t.Errorf("record must name this platform, the version, and an install time, got %+v", record)
	}
}

// realSize asserts the file at path is bytes long.
func realSize(t *testing.T, path string, bytes int64) {
	t.Helper()
	if info, err := os.Stat(path); err != nil || info.Size() != bytes {
		t.Errorf("%s must be recorded at its real size, got %d bytes", filepath.Base(path), bytes)
	}
}
