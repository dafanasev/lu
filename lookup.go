package main

import (
	"fmt"
	"sort"
	"strings"

	yd "github.com/dafanasev/go-yandex-dictionary"
)

func (lu *lu) lookupCycle(ch chan *entry) {
	for lu.scanner.Scan() {
		req := strings.TrimSpace(lu.scanner.Text())
		if req != "" {
			entry := &entry{Request: req}
			for _, lang := range lu.opts.ToLangs {
				translations := lu.lookup(req, lang)
				resp := &response{Lang: lang, Translations: translations}
				entry.Responses = append(entry.Responses, resp)
			}
			// TODO: add to test
			lu.history = append(lu.history, entry)
			ch <- entry
		}
	}
	close(ch)
}

func (lu *lu) lookup(req string, lang string) []string {
	dictResp, err := lu.Lookup(&yd.Params{Lang: lu.opts.FromLang + "-" + lang, Text: req})

	if err == nil {
		var trs []string
		for _, def := range dictResp.Def {
			for _, tr := range def.Tr {
				trs = append(trs, tr.Text)
			}
		}
		return trs
	}

	transResp, err := lu.Translate(lang, req)
	if err != nil || transResp.Result() == req {
		return []string{"no translation"}
	}

	return []string{transResp.Result()}
}

func (lu *lu) supportedLangs(ui string) ([]string, error) {
	resp, err := lu.GetLangs(ui)
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
