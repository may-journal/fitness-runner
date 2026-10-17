package architecture

import (
	"github.com/may-journal/fitness-runner/go/internal/codemodel"
	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/langs"
)

// viewReaders are the language modules' architecture view readers, in
// registry order.
var viewReaders = lang.Views(langs.All)

// referenceReaders are the view readers whose references prove the
// components view's edges, in registry order.
var referenceReaders = withReferences(viewReaders)

func withReferences(readers []codemodel.Reader) []codemodel.Reader {
	var out []codemodel.Reader
	for _, r := range readers {
		if r.References != nil {
			out = append(out, r)
		}
	}
	return out
}

// codeRepo is the repository as the view readers read it.
func (m *Model) codeRepo() codemodel.Repo {
	repo := codemodel.Repo{Files: m.repo.sorted(), Read: m.repo.read}
	for _, s := range m.Sources {
		if s.Category != "test" {
			repo.Code = append(repo.Code, s.Path)
		}
		if s.Category == "product" {
			repo.Product = append(repo.Product, s.Path)
		}
	}
	return repo
}

// language names the language of a file a view reader reads, or "".
func language(file string) string {
	r, _ := codemodel.Of(viewReaders, file)
	return r.Language
}

// declAt is a declaration's first line as a view location.
func declAt(d *codemodel.Decl) Location { return Location(d.At()) }
