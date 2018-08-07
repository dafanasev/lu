package main

import (
	"fmt"
	"sort"
	"strings"

	yd "github.com/dafanasev/go-yandex-dictionary"
)

func (lu *Lu) lookupCycle(done chan struct{}, ch chan *entry) {
	for {
		select {
		case <-done:
			close(ch)
			return
		default:
			if !lu.scanner.Scan() {
				close(ch)
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
				ch <- entry
				lu.history = append(lu.history, entry)
			}
		}
	}
}

func (lu *Lu) lookup(req string, lang string) []string {
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

func (lu *Lu) supportedLangs(ui string) ([]string, error) {
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
