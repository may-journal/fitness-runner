package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/pkg/actions"
)

func init() {
	register("index-key", "Name the checks cache keys base, index and job from the SHA-256 of the release index; an unread index names none.", []string{"base", "index", "job"}, indexKey)
	register("set-key", "Name the key the checks set saves under, first dropping each program an older index's set holds that this run did not link.", []string{"key"}, setKey)
	register("install-source", "Build fitness from the source ref given and put it on PATH, building a merged PR's merge commit when its head is gone.", nil, installSource)
	register("report-action-outcome", "Annotate a failed installation nobody annotated, and write the Fitness action's outcome to the job summary.", nil, reportActionOutcome)
	register("draw-box-diagram", "Draw the box diagram from the performance record with the fitness program in FITNESS, when fitness wrote one.", nil, drawBoxDiagram)
	register("report-outcome", "Annotate a failure fitness did not annotate, and link the complete reports at REPORTS_URL.", nil, reportOutcome)
}

// checkIndex names every check version a release of the checks holds.
const checkIndex = "https://github.com/may-journal/fitness-checks/releases/download/fitness-check-index/index.json"

// sourceModule is every command of this repo, as a source build installs it.
const sourceModule = "github.com/may-journal/fitness-runner/go/cmd/..."

func indexKey(step actions.Step, _ []string) error {
	index, err := actions.Call{}.Raw("curl", "-fsSL", "--retry", "3", checkIndex)
	if err != nil {
		actions.Warning("Could not read the release index, so the checks cache is not restored.")
		return nil
	}
	sum := sha256.Sum256(index)
	base := "fitness-checks-" + env("RUNNER_OS") + "-" + env("RUNNER_ARCH") + "-"
	key := base + hex.EncodeToString(sum[:]) + "-"
	return writeAll(step, "base", base, "index", key, "job", key+env("GITHUB_JOB")+"-")
}

// fitnessCache is the folder fitness caches checks in.
func fitnessCache() string {
	if xdg := env("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "fitness")
	}
	return filepath.Join(env("HOME"), ".cache", "fitness")
}

func setKey(step actions.Step, _ []string) error {
	cache := fitnessCache()
	checks := filepath.Join(cache, "checks")
	if info, err := os.Stat(checks); err != nil || !info.IsDir() {
		return nil
	}
	// A set restored from an older index drops each program this run did not
	// link, so old versions leave the cache.
	if staleSet() {
		pruneUnlinked(checks, filepath.Join(cache, "check-sets"))
	}
	programs := programList(checks)
	if programs == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(programs + "\n"))
	return step.Output("key", env("FITNESS_JOB_KEY")+hex.EncodeToString(sum[:])[:16])
}

// staleSet reports whether the job restored no checks set, or one from
// another release index.
func staleSet() bool {
	restored := env("FITNESS_RESTORED_KEY")
	return restored == "" || !strings.HasPrefix(restored, env("FITNESS_INDEX_KEY"))
}

// pruneUnlinked removes each program folder under checks that no link under
// sets points into.
func pruneUnlinked(checks, sets string) {
	used := map[string]bool{}
	_ = filepath.WalkDir(sets, func(path string, d fs.DirEntry, err error) error {
		if target, linkErr := os.Readlink(path); err == nil && linkErr == nil {
			used[filepath.Dir(filepath.Clean(target))] = true
		}
		return nil
	})
	dirs, _ := filepath.Glob(filepath.Join(checks, "*"))
	for _, dir := range dirs {
		if !used[filepath.Clean(dir)] {
			_ = os.RemoveAll(dir)
		}
	}
}

// programList is each file under checks as `./<folder>/<program>`, sorted,
// one per line, which names the set.
func programList(checks string) string {
	var programs []string
	_ = filepath.WalkDir(checks, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			rel, _ := filepath.Rel(checks, path)
			programs = append(programs, "./"+filepath.ToSlash(rel))
		}
		return nil
	})
	slices.Sort(programs)
	return strings.Join(programs, "\n")
}

func installSource(_ actions.Step, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("name the source ref to build")
	}
	if err := installRef(args[0]); err != nil {
		return err
	}
	gopath, err := actions.Call{}.Capture("go", "env", "GOPATH")
	if err != nil {
		return err
	}
	return actions.Path(filepath.Join(gopath, "bin"))
}

// installRef builds every command at ref. A merged pull request's head is
// gone with its branch, so a failed build retries on its merge commit.
func installRef(ref string) error {
	if actions.Run("go", "install", sourceModule+"@"+ref) == nil {
		return nil
	}
	merge, number := mergeCommitFor(ref)
	if merge == "" {
		return actions.Fail("go install of " + ref + " failed, and it is not the head of a merged fitness-runner pull request, so no other commit stands in for it.")
	}
	actions.Notice(fmt.Sprintf("Pull request %s merged and its head %s is gone, so fitness builds from its merge commit %s.", number, ref, merge))
	return actions.Run("go", "install", sourceModule+"@"+merge)
}

// mergeCommitFor finds the fitness-runner pull request the event names. It
// returns its number and, when it merged with head ref, its merge commit.
func mergeCommitFor(ref string) (merge, number string) {
	number = eventNumber()
	if number == "" || env("GITHUB_REPOSITORY") != "may-journal/fitness-runner" {
		return "", number
	}
	return mergedHead(number, ref), number
}

// mergedHead is the merge commit of pull request number when it merged
// with head ref, or "".
func mergedHead(number, ref string) string {
	var pull struct {
		Merged bool `json:"merged"`
		Head   struct {
			SHA string `json:"sha"`
		} `json:"head"`
		MergeCommit string `json:"merge_commit_sha"`
	}
	if actions.GhJSON(&pull, "api", pullRequest{number}.path()) != nil || !pull.Merged || pull.Head.SHA != ref {
		return ""
	}
	return pull.MergeCommit
}

// eventNumber is the pull request or issue number the event names, or "".
func eventNumber() string {
	raw, _ := os.ReadFile(env("GITHUB_EVENT_PATH"))
	var event struct {
		PullRequest struct {
			Number json.Number `json:"number"`
		} `json:"pull_request"`
		Issue struct {
			Number json.Number `json:"number"`
		} `json:"issue"`
	}
	_ = json.Unmarshal(raw, &event)
	return string(event.PullRequest.Number) + string(event.Issue.Number)
}

func reportActionOutcome(actions.Step, []string) error {
	status := env("REPORT_STATUS")
	if status == "failure" && env("REPORT_ALREADY_ANNOTATED") != "true" {
		actions.Error("Fitness installation or execution failed; open the action logs for the failed operation and its cause.")
		if err := actions.Env("FITNESS_FAILURE_REPORTED", "true"); err != nil {
			return err
		}
	}
	return actions.Summary(fmt.Sprintf("### Fitness action: %s\n\n[View steps and logs](%s/%s/actions/runs/%s)", status, env("GITHUB_SERVER_URL"), env("GITHUB_REPOSITORY"), env("GITHUB_RUN_ID")))
}

func drawBoxDiagram(actions.Step, []string) error {
	record := performanceRecord("fitness.json")
	if !exists(record) {
		return nil
	}
	return actions.Run(env("FITNESS"), "boxes", record, performanceRecord("fitness.html"))
}

func reportOutcome(actions.Step, []string) error {
	if env("JOB_STATUS") == "failure" && env("ALREADY_ANNOTATED") != "true" {
		actions.Error("Workflow job " + env("GITHUB_JOB") + " failed; open its logs for the failed step and cause.")
	}
	if url := env("REPORTS_URL"); url != "" {
		return actions.Summary(fmt.Sprintf("[Download complete Fitness reports](%s)", url))
	}
	return nil
}
