package main

import (
	yd "github.com/dafanasev/go-yandex-dictionary"
	yt "github.com/dafanasev/go-yandex-translate"
	"github.com/pkg/errors"
)

// dictionaryMock is the mock for the dictionary interface,
// used for tests and debug purposes
type dictionaryMock struct{}

func (m *dictionaryMock) Lookup(params *yd.Params) (*yd.Entry, error) {
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

// translatorMock is the mock for the translator interface,
// used for tests and debug purposes
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
