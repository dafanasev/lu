package main

import (
	"fmt"
	"sort"
	"strings"

	yd "github.com/dafanasev/go-yandex-dictionary"
)

// lookupCycle iterates through data source line by line,
// making look ups for all needed languages for non empty lines
// adding results wrapped into entries struct to the history list and
// passing them to the corresponding channel.
// The cycle can be stopped at any moment using done channel
func (lu *Lu) lookupCycle(done chan struct{}, entriesCh chan *entry) {
	for {
		select {
		case <-done:
			close(entriesCh)
			return
		default:
			if !lu.scanner.Scan() {
				close(entriesCh)
				return
			}

			req := strings.TrimSpace(lu.scanner.Text())
			if req != "" {
				entry := &entry{Request: req}
				for _, lang := range lu.opts.ToLangs {
					translations := lu.lookup(req, lang)
					resp := &response{Lang: lang, Translations: translations}
					entry.Responses = append(entry.Responses, resp)
				}
				entriesCh <- entry
				lu.history = append(lu.history, entry)
			}
		}
	}
}

// lookup returns results of the call to dictionary and,
// if there are no ones, to translator
// It returns "no translation" if the call to translator returns no results too
func (lu *Lu) lookup(req string, lang string) []string {
	dictResp, err := lu.dictionary.Lookup(&yd.Params{Lang: lu.opts.FromLang + "-" + lang, Text: req})

	if err == nil {
		var trs []string
		// iterating through yandex dictionary data structures
		// to accumulate all definitions in a list and return it
		for _, def := range dictResp.Def {
			for _, tr := range def.Tr {
				trs = append(trs, tr.Text)
			}
		}
		return trs
	}

	transResp, err := lu.translator.Translate(lang, req)
	// translator returns request string as the result if there is no translation
	if err != nil || transResp.Result() == req {
		return []string{"no translation"}
	}

	return []string{transResp.Result()}
}

// supportedLangs returns the list of the languages supported by Yandex APIs
func (lu *Lu) supportedLangs(ui string) ([]string, error) {
	resp, err := lu.translator.GetLangs(ui)
	if err != nil {
		return nil, err
	}
	var langs []string
	for abbr, lang := range resp.Langs {
		langs = append(langs, fmt.Sprintf("%s: %s", abbr, lang))
	}
	sort.Strings(langs)

	return langs, nil
}
