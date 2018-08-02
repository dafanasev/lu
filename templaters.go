package main

import "github.com/gobuffalo/packr"

var box = packr.NewBox("templates")

type templater interface {
	entry() string
	list() string
}

type layoutTemplater interface {
	layout() string
}

type textTemplater struct{}

func (t *textTemplater) entry() string {
	return box.String("entry.text.tmpl")
}

func (t *textTemplater) list() string {
	return box.String("list.text.tmpl")
}

type htmlTemplater struct{}

func (t *htmlTemplater) entry() string {
	return box.String("entry.html.tmpl")
}

func (t *htmlTemplater) list() string {
	return box.String("list.html.tmpl")
}

func (t *htmlTemplater) layout() string {
	return box.String("layout.html.tmpl")
}

func inc(i int) int {
	return i + 1
}

func dict(values ...interface{}) map[string]interface{} {
	d := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		d[values[i].(string)] = values[i+1]
	}
	return d
}
