package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_textTemplater_list(t *testing.T) {
	assert.Contains(t, (&textTemplater{}).list(), "{{ range .Entries -}}")
}

func Test_textTemplater_entry(t *testing.T) {
	assert.Contains(t, (&textTemplater{}).entry(), "{{ range $idx, $tr := .Translations -}}")
}

func Test_htmlTemplater_layout(t *testing.T) {
	assert.Contains(t, (&htmlTemplater{}).layout(), "<meta charset=\"utf-8\">")
}

func Test_htmlTemplater_list(t *testing.T) {
	assert.Contains(t, (&htmlTemplater{}).list(), "<ol id=\"req-list\">")
}

func Test_htmlTemplater_entry(t *testing.T) {
	assert.Contains(t, (&htmlTemplater{}).entry(), "<header>{{ .Lang }}</header>")
}

func Test_templatesFnMap_inc(t *testing.T) {
	incFn := templatesFnMap["inc"].(func(int) int)
	assert.Equal(t, incFn(2), 3)
}

func Test_templatesFnMap_dict(t *testing.T) {
	dictFn := templatesFnMap["dict"].(func(values ...interface{}) map[string]interface{})
	args := []interface{}{"one", 1, "two", "2", "three", "drei"}
	expected := map[string]interface{}{"one": 1, "two": "2", "three": "drei"}
	assert.Equal(t, expected, dictFn(args...))

	args = append(args, "odd")
	assert.Panics(t, func() { dictFn(args...) })
}
