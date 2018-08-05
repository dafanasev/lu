package main

import (
	"html/template"

	"github.com/gobuffalo/packr"
)

var box = packr.NewBox("templates")

var templaesFnMap = template.FuncMap{
	"inc": func(i int) int {
		return i + 1
	},
	"dict": func(values ...interface{}) map[string]interface{} {
		d := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			d[values[i].(string)] = values[i+1]
		}
		return d
	},
}

type templater interface {
	entry() string
	list() string
}

type layoutTemplater interface {
	layout() string
}

type textTemplater struct{}

func (t *textTemplater) list() string {
	return box.String("list.text.tmpl")
}

func (t *textTemplater) entry() string {
	return box.String("entry.text.tmpl")
}

type htmlTemplater struct{}

func (t *htmlTemplater) layout() string {
	return box.String("layout.html.tmpl")
}

func (t *htmlTemplater) list() string {
	return box.String("list.html.tmpl")
}

func (t *htmlTemplater) entry() string {
	return box.String("entry.html.tmpl")
}
