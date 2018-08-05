package main

import (
	"fmt"
	"sort"
	"strings"

	yd "github.com/dafanasev/go-yandex-dictionary"
	yt "github.com/dafanasev/go-yandex-translate"
)

type dictionary interface {
	Lookup(params *yd.Params) (*yd.Entry, error)
}

type translator interface {
	Translate(lang, text string) (*yt.Response, error)
	GetLangs(ui string) (*yt.Languages, error)
}

func lookupCycle(ch chan *entry) {
	for scanner.Scan() {
		req := strings.TrimSpace(scanner.Text())
		if req != "" {
			entry := &entry{Request: req}
			for _, lang := range opts.ToLangs {
				translations := lookup(req, lang)
				resp := &response{Lang: lang, Translations: translations}
				entry.Responses = append(entry.Responses, resp)
			}
			ch <- entry
		}
	}
	close(ch)
}

func lookup(req string, lang string) []string {
	dictResp, err := dict.Lookup(&yd.Params{Lang: opts.FromLang + "-" + lang, Text: req})

	if err == nil {
		var trs []string
		for _, def := range dictResp.Def {
			for _, tr := range def.Tr {
				trs = append(trs, tr.Text)
			}
		}
		return trs
	}

	transResp, err := tr.Translate(lang, req)
	if err != nil || transResp.Result() == req {
		return []string{"no translation"}
	}

	return []string{transResp.Result()}
}

func supportedLangs(ui string) ([]string, error) {
	resp, err := tr.GetLangs(ui)
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
