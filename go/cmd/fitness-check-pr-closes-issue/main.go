package check

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

func Main() {
	if emitClosedMode(os.Args[1:]) {
		return
	}
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "pr-closes-issue", Scope: checkkit.ScopeBlob, Blob: true, ContextInlineArg: "--body"},
		Run:      run,
	})
}

// closingKeywords is GitHub's fixed set of nine issue-closing keywords, shared by the Plan side through --emit-closed and the PR side.
const closingKeywords = `close|closes|closed|fix|fixes|fixed|resolve|resolves|resolved`

// closingIssueRe matches a closing keyword linking an issue number, e.g.
// `Closes #12` or `Fixes: #12`.
var closingIssueRe = regexp.MustCompile(`(?i)\b(?:` + closingKeywords + `)[:\s]+#(\d+)`)

// implementedIssueRe matches an `Implements #12` or `Plan #12` claim, which needs a closing keyword for that same issue.
var implementedIssueRe = regexp.MustCompile(`(?i)\b(?:implement(?:s|ed)?|plan)[:\s]+#(\d+)`)

func run(_ string, args []string) (checkkit.Result, error) {
	required := parseRequireClose(args)
	return bodycheck.RunIf(args, bodycheck.PRBody, func(body string) []string {
		return validate(body, required)
	})
}

// validate reports the PR body's closure violations, including any issue of a closed Plan that the PR leaves open.
func validate(body string, required []int) []string {
	body = mdx.StripCode(body)
	closed := closedIssues(body)
	errors := baseErrors(body, closed)
	for _, n := range required {
		if !closed[n] {
			errors = append(errors, fmt.Sprintf(
				"PR closes a Plan that closes #%d, so the PR must also close #%d — add a closing keyword for it", n, n))
		}
	}
	return errors
}

// baseErrors reports the two body-only rules: at least one closing keyword,
// and every implemented issue closed.
func baseErrors(body string, closed map[int]bool) []string {
	var errors []string
	if !closingIssueRe.MatchString(body) {
		errors = append(errors,
			"PR description has no GitHub closing keyword (close/fix/resolve + #NN) — every PR must close at least one issue on merge")
	}
	for _, n := range implementedIssues(body) {
		if !closed[n] {
			errors = append(errors, fmt.Sprintf(
				"PR says it implements #%d but no closing keyword (close/fix/resolve + #%d) closes it", n, n))
		}
	}
	return errors
}

// emitClosedMode prints the body's closed issue numbers as a JSON array for --emit-closed and reports it handled.
func emitClosedMode(argv []string) bool {
	if !hasFlag(argv, "--emit-closed") {
		return false
	}
	body, _, err := bodycheck.BodyFile(argv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := json.NewEncoder(os.Stdout).Encode(sortedClosed(closedIssues(mdx.StripCode(body)))); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	return true
}

func hasFlag(argv []string, flag string) bool {
	for _, a := range argv {
		if a == flag {
			return true
		}
	}
	return false
}

// closedIssues is the set of issue numbers the body closes with a keyword.
func closedIssues(body string) map[int]bool {
	set := map[int]bool{}
	for _, m := range closingIssueRe.FindAllStringSubmatch(body, -1) {
		if n, err := strconv.Atoi(m[1]); err == nil {
			set[n] = true
		}
	}
	return set
}

// sortedClosed returns the set's numbers in ascending order.
func sortedClosed(set map[int]bool) []int {
	nums := make([]int, 0, len(set))
	for n := range set {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	return nums
}

// implementedIssues lists, in first-seen order without duplicates, the issue
// numbers the body claims to implement.
func implementedIssues(body string) []int {
	var nums []int
	seen := map[int]bool{}
	for _, m := range implementedIssueRe.FindAllStringSubmatch(body, -1) {
		n, err := strconv.Atoi(m[1])
		if err != nil || seen[n] {
			continue
		}
		seen[n] = true
		nums = append(nums, n)
	}
	return nums
}

// parseRequireClose reads --require-close=NN,NN (or the space form) into the
// issue numbers the PR must close because a Plan it closes closes them.
func parseRequireClose(args []string) []int {
	return parseIntList(requireCloseRaw(args))
}

func requireCloseRaw(args []string) string {
	for i, a := range args {
		if v, ok := strings.CutPrefix(a, "--require-close="); ok {
			return v
		}
		if a == "--require-close" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func parseIntList(raw string) []int {
	var nums []int
	for _, part := range strings.Split(raw, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			nums = append(nums, n)
		}
	}
	return nums
}
