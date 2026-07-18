package checkkit

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	root, args, describe := parseArgs([]string{"--root", "/repo", "--write", "."})
	if root != "/repo" || describe || !reflect.DeepEqual(args, []string{"--write", "."}) {
		t.Fatalf("got root=%q args=%v describe=%v", root, args, describe)
	}
	root, args, describe = parseArgs([]string{"--root=/x", "--describe"})
	if root != "/x" || !describe || args != nil {
		t.Fatalf("got root=%q args=%v describe=%v", root, args, describe)
	}
}

func TestPassFailShapes(t *testing.T) {
	p := Pass(3)
	if !p.Ok || p.FilesChecked != 3 || len(p.Errors) != 0 {
		t.Fatalf("pass: %+v", p)
	}
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
