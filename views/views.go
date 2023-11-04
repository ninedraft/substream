package views

import (
	"embed"
	"html/template"
	"io"
)

type View[E any] string

//go:embed *.html
var assets embed.FS

var tmpls = template.Must(template.ParseFS(assets, "*.html"))

func (view View[E]) Render(dst io.Writer, args *E) error {
	return tmpls.ExecuteTemplate(dst, string(view), args)
}
