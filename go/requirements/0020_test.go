package requirements

import (
	"os"
	"path/filepath"
	"testing"
)

// commitAttributionTrailers is a commit message carrying both AI trailers.
const commitAttributionTrailers = "feat(api): add endpoint\n\nAI-Tools: Claude Code\nAI-Models: Opus 4.8\n"

// commitAttributionMissing is the error for a missing trailer key.
func commitAttributionMissing(key string) string {
	return `commit message missing "` + key + `:" trailer`
}

// commitAttribution commits each message on happyRepo in order, the last at
// HEAD, then runs `fitness-install -- commit-attribution <args>`.
func commitAttribution(t *testing.T, messages []string, args ...string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", nil)
	for _, m := range messages {
		commit(t, repo, nil, m)
	}
	return fitness(t, repo, nil, append([]string{"commit-attribution"}, args...)...)
}

func Test0020_1(t *testing.T) {
	out, code := commitAttribution(t, []string{"feat(api): add endpoint\n\nNo trailers here.\n"})
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "commit-attribution")

	folder := example(t, "happyRepo", nil)
	mustDo(t, os.RemoveAll(filepath.Join(folder, ".git")))
	out, code = fitness(t, folder, nil, "commit-attribution")
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "commit-attribution")
}

func Test0020_2(t *testing.T) {
	out, code := commitAttribution(t, []string{commitAttributionTrailers})
	sees(t, out, code, 0, "All 1 checks passed · 1 files scanned")
}

func Test0020_3(t *testing.T) {
	out, code := commitAttribution(t, []string{commitAttributionTrailers, "feat(api): add paging\n\nJust a body.\n"})
	sees(t, out, code, 1, commitAttributionMissing("AI-Models"), commitAttributionMissing("AI-Tools"))
}

func Test0020_4(t *testing.T) {
	out, code := commitAttribution(t, []string{commitAttributionTrailers}, "--message=feat(api): add paging\n\nJust a body.\n")
	sees(t, out, code, 1, commitAttributionMissing("AI-Models"), commitAttributionMissing("AI-Tools"))
}

func Test0020_5(t *testing.T) {
	out, code := commitAttribution(t, []string{commitAttributionTrailers}, "--message=")
	sees(t, out, code, 1, `No commit message to validate; add "AI-Tools:" and "AI-Models:" trailers to disclose AI usage`)
}

func Test0020_6(t *testing.T) {
	for _, subject := range []string{"Merge branch 'feature' into main", `Revert "feat(api): add endpoint"`} {
		out, code := commitAttribution(t, []string{commitAttributionTrailers, subject})
		sees(t, out, code, 0, "All 1 checks passed")
	}
}

func Test0020_7(t *testing.T) {
	out, code := commitAttribution(t, []string{commitAttributionTrailers, "feat(api): add paging\n\nAI-Tools: Claude Code\nAI-Models:   \n"})
	sees(t, out, code, 1, commitAttributionMissing("AI-Models"))
}

func Test0020_8(t *testing.T) {
	out, code := commitAttribution(t, []string{commitAttributionTrailers, "feat(api): add paging\n\nAI-Tools: Claude Code\nWe set AI-Models: Opus 4.8 later.\n"})
	sees(t, out, code, 1, commitAttributionMissing("AI-Models"))
}
