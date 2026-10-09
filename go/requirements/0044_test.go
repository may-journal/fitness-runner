package requirements

import (
	"strings"
	"testing"
)

// reframing runs `fitness-install -- no-contrastive-reframing` on happyRepo
// with prose appended to its README.
func reframing(t *testing.T, prose string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", map[string]string{"README.md": readme + "\n" + prose})
	return fitness(t, repo, nil, "no-contrastive-reframing")
}

// reframingHint is the start of every failure message.
const reframingHint = `README.md: rewrite as a direct statement, not the "not X, it's Y" pattern: `

func Test0044_1(t *testing.T) {
	t.Parallel()
	out, code := reframing(t, "It's not a workout. It's a lifestyle.\n\nThey’re not bugs.\n\nThey’re features.\n")
	sees(t, out, code, 1, reframingHint+`"It's not a workout It's a lifestyle"`,
		reframingHint+`"They’re not bugs They’re features"`)
}

func Test0044_2(t *testing.T) {
	t.Parallel()
	out, code := reframing(t, "This isn't about speed; it's about consistency.\n")
	sees(t, out, code, 1, reframingHint+`"This isn't about speed; it's about consistency"`)
}

func Test0044_3(t *testing.T) {
	t.Parallel()
	out, code := reframing(t, "It's not ready yet. The file is not found, but the fallback loads.\n\n"+
		"We should not merge, but wait for review. Not only fast, but also correct.\n\n"+
		"It's not X. It's not Y.\n")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0044_4(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{
		"README.md": "---\ntitle: It's not X. It's Y.\n---\n\n# App\n\n```\nIt's not a workout. It's a lifestyle.\n```\n\n" +
			"## It's not a workout. It's a lifestyle.\n\n" +
			"Use `It's not a workout. It's a lifestyle.` verbatim.\n",
	})
	out, code := fitness(t, repo, nil, "no-contrastive-reframing")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0044_5(t *testing.T) {
	t.Parallel()
	out, code := reframing(t, `The phrase "It's not a workout. It's a lifestyle." is banned.`+"\n\n"+
		"The phrase “It's not a workout. It's a lifestyle.” is banned.\n")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0044_6(t *testing.T) {
	t.Parallel()
	long := "It's not " + strings.Repeat("a very long clause ", 10) + ", but simple."
	out, code := reframing(t, long+"\n")
	sees(t, out, code, 1, reframingHint+`"`+long[:80]+`…"`)
}
