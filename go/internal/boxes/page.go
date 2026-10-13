package boxes

import (
	_ "embed"
	"html/template"
	"io"
)

//go:embed page.html.tmpl
var pageSource string

var page = template.Must(template.New("page").Funcs(template.FuncMap{
	"ms":    formatMs,
	"bytes": formatBytes,
}).Parse(pageSource))

// pageData is what the template draws.
type pageData struct {
	Report
	Passed, Failed int
	Views          []view
}

// Write renders r as one self-contained HTML page: no scripts, fonts, or
// requests, so it opens straight from a downloaded artifact.
func Write(w io.Writer, r Report) error {
	data := pageData{Report: r}
	for _, c := range r.Checks {
		if c.Ok {
			data.Passed++
		} else {
			data.Failed++
		}
	}
	for _, m := range measures {
		data.Views = append(data.Views, layoutView(r.Checks, m))
	}
	return page.Execute(w, data)
}
