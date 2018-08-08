package main

import (
	"html/template"

	"github.com/gobuffalo/packr"
)

// box used to embed templates into executable binary
var box = packr.NewBox("templates")

// templatesFnMap holds functions use in templates
var templatesFnMap = template.FuncMap{
	"inc": func(i int) int {
		return i + 1
	},
	// dict used to pass multiple values (in a map) to partial template
	"dict": func(values ...interface{}) map[string]interface{} {
		d := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			d[values[i].(string)] = values[i+1]
		}
		return d
	},
}

// templater methods return templates (as strings) used to render lookup results
type templater interface {
	entry() string
	list() string
}

// layoutTemplater is the optional interface
// that defines the Layout method used to render output file layout
type layoutTemplater interface {
	layout() string
}

// textTemplater implements templater interface to print lookup results to stdout and render text files
type textTemplater struct{}

// list returns list text template from the box, which, in turn loads it from the FS
// and embeds in the executable binary
func (t *textTemplater) list() string {
	return box.String("list.text.tmpl")
}

func (t *textTemplater) entry() string {
	return box.String("entry.text.tmpl")
}

// htmlTemplater implements templater and layoutTemplater interfaces to render lookup results to html files
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
