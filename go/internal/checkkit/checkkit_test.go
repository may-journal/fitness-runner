package checkkit

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	cases := []struct {
		in           []string
		wantRoot     string
		wantArgs     []string
		wantDescribe bool
	}{
		{[]string{"--root", "/repo", "--write", "."}, "/repo", []string{"--write", "."}, false},
		{[]string{"--root=/x", "--describe"}, "/x", nil, true},
	}
	for _, tc := range cases {
		root, args, describe := parseArgs(tc.in)
		if root != tc.wantRoot || describe != tc.wantDescribe || !reflect.DeepEqual(args, tc.wantArgs) {
			t.Fatalf("parseArgs(%v): got root=%q args=%v describe=%v", tc.in, root, args, describe)
		}
	}
}

func TestPassShape(t *testing.T) {
	p := Pass(3)
	if !p.Ok || p.FilesChecked != 3 || len(p.Errors) != 0 {
		t.Fatalf("pass: %+v", p)
	}
}

func TestFailShape(t *testing.T) {
	f := Fail(1, "a", "b")
	if f.Ok || !reflect.DeepEqual(f.Errors, []string{"a", "b"}) {
		t.Fatalf("fail: %+v", f)
	}
}

func TestEnvAccessors(t *testing.T) {
	t.Setenv("FITNESS_STAGED_FILES", "a.txt\nb/c.md")
	if got := StagedFiles(); !reflect.DeepEqual(got, []string{"a.txt", "b/c.md"}) {
		t.Fatalf("staged: %v", got)
	}
	t.Setenv("FITNESS_STAGED_FILES", "")
	if got := StagedFiles(); got != nil {
		t.Fatalf("empty staged should be nil, got %v", got)
	}
	t.Setenv("FITNESS_CTX_MESSAGE", "")
	if msg, ok := CtxMessage(); !ok || msg != "" {
		t.Fatal("present-but-empty message must report ok=true")
	}
}
