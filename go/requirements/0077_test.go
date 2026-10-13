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
	if cspell := reported(t, report, "cspell"); cspell.Ok || cspell.FilesChecked < 1 {
		t.Errorf("cspell must fail having judged files, got %+v", cspell)
	}
	if prose := reported(t, report, "prose-budget"); !prose.Ok {
		t.Errorf("prose-budget must pass, got %+v", prose)
	}
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
