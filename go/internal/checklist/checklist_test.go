package checklist

import (
	"reflect"
	"testing"
)

func TestUnchecked(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []string
	}{
		{"none", "no list here", nil},
		{"all ticked", "- [x] a\n- [X] b", nil},
		{"mixed", "- [x] done\n- [ ] open one\n* [ ] open two\n  + [ ] nested\n1. [ ] numbered", []string{"open one", "open two", "nested", "numbered"}},
		{"crlf", "- [ ] a\r\n- [ ] b\r\n", []string{"a", "b"}},
		{"empty item is not a task", "- [ ] \n- [ ]", nil},
		{"backtick fence", "```md\n- [ ] example\n```\n- [ ] real", []string{"real"}},
		{"tilde fence", "~~~\n- [ ] example\n~~~\n- [ ] real", []string{"real"}},
		{"shorter closer stays open", "````\n- [ ] a\n```\n- [ ] b\n````\n- [ ] c", []string{"c"}},
		{"other marker stays open", "```\n~~~\n- [ ] a\n```\n- [ ] b", []string{"b"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Unchecked(tc.body); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Unchecked = %q, want %q", got, tc.want)
			}
		})
	}
}
