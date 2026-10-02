package walkfs

import (
	"reflect"
	"testing"
)

func TestWithoutGenerated(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q")
	write(t, root, ".gitattributes", "*.lock linguist-generated\npackage-lock.json linguist-generated=true\nkeep.json linguist-generated=false\n")
	files := []string{"a.md", "deps/x.lock", "package-lock.json", "keep.json"}
	got := WithoutGenerated(root, files)
	if want := []string{"a.md", "keep.json"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("WithoutGenerated = %v, want %v", got, want)
	}
}

func TestWithoutGeneratedOutsideGit(t *testing.T) {
	root := t.TempDir()
	write(t, root, ".gitattributes", "*.lock linguist-generated\n")
	files := []string{"x.lock"}
	if got := WithoutGenerated(root, files); !reflect.DeepEqual(got, files) {
		t.Fatalf("outside git = %v, want every file kept", got)
	}
	if got := WithoutGenerated(root, nil); got != nil {
		t.Fatalf("no files = %v, want nil", got)
	}
}
