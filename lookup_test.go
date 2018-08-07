package main

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Lu_lookup(t *testing.T) {
	lu := &Lu{opts: options{FromLang: "en"}}
	lu.dictionary = &dictionaryMock{}
	lu.translator = &translatorMock{}

	assert.Equal(t, []string{"Hund", "Rüde", "geiler Bock"}, lu.lookup("dog", "de"))
	assert.Equal(t, []string{"schwarzer Hund"}, lu.lookup("black dog", "de"))
	assert.Equal(t, []string{"no translation"}, lu.lookup("cat", "de"))
	assert.Equal(t, []string{"no translation"}, lu.lookup("black dog", "fr"))
}

func Test_Lu_lookupCycle(t *testing.T) {
	lu := &Lu{opts: options{FromLang: "en", ToLangs: []string{"de"}}}
	lu.dictionary = &dictionaryMock{}
	lu.translator = &translatorMock{}

	s := `
	dog
	black dog
	
	cat
	`
	lu.scanner = bufio.NewScanner(strings.NewReader(s))

	done := make(chan struct{})
	ch := make(chan *entry)
	close(done)
	go lu.lookupCycle(done, ch)
	assert.Equal(t, 0, len(lu.history))

	expected := map[string][]string{
		"dog":       {"Hund", "Rüde", "geiler Bock"},
		"black dog": {"schwarzer Hund"},
		"cat":       {"no translation"},
	}

	done = make(chan struct{})
	ch = make(chan *entry)
	go lu.lookupCycle(done, ch)

	var entries []*entry
	for entry := range ch {
		entries = append(entries, entry)
		translations, ok := expected[entry.Request]
		require.True(t, ok)
		assert.Equal(t, translations, entry.Responses[0].Translations)
		assert.Equal(t, "de", entry.Responses[0].Lang)
	}
	assert.Equal(t, 3, len(entries))
	assert.Equal(t, 3, len(lu.history))
}

func Test_Lu_supportedLangs(t *testing.T) {
	lu := &Lu{}
	lu.translator = &translatorMock{}

	_, err := lu.supportedLangs("")
	require.Error(t, err)

	resp, err := lu.supportedLangs("en")
	require.NoError(t, err)
	assert.Equal(t, []string{"de: german", "en: english", "it: italian"}, resp)
}
