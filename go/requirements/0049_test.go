package requirements

import "testing"

// planTrailer runs `fitness-install -- plan-trailer --message <msg>` on
// happyRepo, as the commit-msg hook passes a proposed message.
func planTrailer(t *testing.T, msg string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", nil), nil, "plan-trailer", "--message", msg)
}

// planTrailerFix is the hint every misspelled plan reference carries.
const planTrailerFix = `must read exactly "Plan #<number>"`

func Test0049_1(t *testing.T) {
	t.Parallel()
	out, code := planTrailer(t, "feat(app): add a feature\n\nSome body.\n")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0049_2(t *testing.T) {
	t.Parallel()
	out, code := planTrailer(t, "feat(app): add a feature\n\nSome body.\n\nPlan #63\n")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0049_3(t *testing.T) {
	t.Parallel()
	out, code := planTrailer(t, "feat(app): add a feature\n\nPlan 63\nplan #63\nPlan: 63\n")
	sees(t, out, code, 1,
		`plan trailer "Plan 63" `+planTrailerFix,
		`plan trailer "plan #63" `+planTrailerFix,
		`plan trailer "Plan: 63" `+planTrailerFix)
}

func Test0049_4(t *testing.T) {
	t.Parallel()
	out, code := planTrailer(t, "feat(app): add a feature\n\nPlan the rollout in 3 steps.\n")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0049_5(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	commit(t, repo, nil, "feat(app): add a feature\n\nPlan: 63")
	out, code := fitness(t, repo, nil, "plan-trailer")
	sees(t, out, code, 1, `plan trailer "Plan: 63" `+planTrailerFix)
}
