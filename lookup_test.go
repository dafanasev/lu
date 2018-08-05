package main

import (
	"bufio"
	"strings"
	"testing"

	yd "github.com/dafanasev/go-yandex-dictionary"
	yt "github.com/dafanasev/go-yandex-translate"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type DictionaryMock struct{}

func (m *DictionaryMock) Lookup(params *yd.Params) (*yd.Entry, error) {
	if params.Text == "dog" && params.Lang == "en-de" {
		var trs1 []yd.Tr
		trs1 = append(trs1, yd.Tr{Text: "Hund"})
		trs1 = append(trs1, yd.Tr{Text: "Rüde"})

		trs2 := []yd.Tr{{Text: "geiler Bock"}}

		defs := []yd.Def{{Tr: trs1}, {Tr: trs2}}
		return &yd.Entry{Def: defs}, nil
	}
	return nil, errors.New("no entry")
}

type translatorMock struct{}

func (m *translatorMock) Translate(lang, text string) (*yt.Response, error) {
	if text == "black dog" && lang == "de" {
		return &yt.Response{Text: []string{"schwarzer Hund"}}, nil
	}
	return nil, errors.New("no translation")
}

func (m *translatorMock) GetLangs(ui string) (*yt.Languages, error) {
	if ui == "en" {
		return &yt.Languages{Langs: map[string]string{"en": "english", "de": "german", "it": "italian"}}, nil
	}
	return nil, errors.New("wrong lang")
}

func Test_lookup(t *testing.T) {
	dict = &DictionaryMock{}
	tr = &translatorMock{}
	opts.FromLang = "en"

	assert.Equal(t, []string{"Hund", "Rüde", "geiler Bock"}, lookup("dog", "de"))
	assert.Equal(t, []string{"schwarzer Hund"}, lookup("black dog", "de"))
	assert.Equal(t, []string{"no translation"}, lookup("cat", "de"))
	assert.Equal(t, []string{"no translation"}, lookup("black dog", "fr"))
}

func Test_lookupCycle(t *testing.T) {
	dict = &DictionaryMock{}
	tr = &translatorMock{}
	opts.FromLang = "en"
	opts.ToLangs = []string{"de"}

	s := `
	dog
	black dog
	
	cat
	`
	scanner = bufio.NewScanner(strings.NewReader(s))

	ch := make(chan *entry)
	go lookupCycle(ch)

	expected := map[string][]string{
		"dog":       {"Hund", "Rüde", "geiler Bock"},
		"black dog": {"schwarzer Hund"},
		"cat":       {"no translation"},
	}

	var entries []*entry
	for entry := range ch {
		entries = append(entries, entry)
		translations, ok := expected[entry.Request]
		require.True(t, ok)
		assert.Equal(t, translations, entry.Responses[0].Translations)
		assert.Equal(t, "de", entry.Responses[0].Lang)
	}
	assert.Equal(t, 3, len(entries))
}

func Test_supportedLangs(t *testing.T) {
	tr = &translatorMock{}

	_, err := supportedLangs("")
	require.Error(t, err)

	resp, err := supportedLangs("en")
	require.NoError(t, err)
	assert.Equal(t, []string{"de: german", "en: english", "it: italian"}, resp)
}
