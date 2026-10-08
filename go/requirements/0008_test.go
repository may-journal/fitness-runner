package requirements

import "testing"

func Test0008_1(t *testing.T) {
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "no-such-check")
	sees(t, out, code, 1, "Unknown check: no-such-check")
}

func Test0008_2(t *testing.T) {
	out, code := fitness(t, example(t, "happyRepo", map[string]string{".fitnessrc.json": "{\n"}), nil)
	sees(t, out, code, 1, "fitness setup: .fitnessrc.json:")
}

func Test0008_3(t *testing.T) {
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "--help")
	sees(t, out, code, 0, "fitness — run the configured checks.", "fitness init")
}
