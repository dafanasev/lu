package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"os"
	"sort"

	yd "github.com/dafanasev/go-yandex-dictionary"
	yt "github.com/dafanasev/go-yandex-translate"
)

type lu struct {
	dictionary
	translator
	opts          options
	scanner       *bufio.Scanner
	fileTemplater templater
	srcFile       *os.File
	dstFile       *os.File
	history       []*entry
	ch            chan *entry
}

type dictionary interface {
	Lookup(params *yd.Params) (*yd.Entry, error)
}

type translator interface {
	Translate(lang, text string) (*yt.Response, error)
	GetLangs(ui string) (*yt.Languages, error)
}

// entry holds request and corresponding responses, one for each requested language
type entry struct {
	Request   string
	Responses []*response
}

// response is the single response
type response struct {
	Lang         string
	Translations []string
}

type entriesByReq []*entry

func (br entriesByReq) Len() int           { return len(br) }
func (br entriesByReq) Swap(i, j int)      { br[i], br[j] = br[j], br[i] }
func (br entriesByReq) Less(i, j int) bool { return br[i].Request < br[j].Request }

func newLu(args []string, opts options) (*lu, error) {
	lu := &lu{opts: opts}

	err := lu.setup(args)
	if err != nil {
		return nil, err
	}

	lu.ch = make(chan *entry)
	go lu.lookupCycle(lu.ch)

	return lu, nil
}

func (lu *lu) shouldPrintEntry() bool {
	// print to stdout if there is no destination file - i.e. destination is stdout
	// or if there is no source file, because in this case source is stdin
	// and we want to see output in the terminal too, even if the destination file is specified
	// otherwise show progress
	return lu.srcFile == nil || lu.dstFile == nil
}

func (lu *lu) showLangs() error {
	langs, err := lu.supportedLangs("en")
	if err != nil {
		return err
	}
	for _, lang := range langs {
		fmt.Println(lang)
	}
	return nil
}

func (lu *lu) cleanUp() error {
	if lu.srcFile != nil {
		lu.srcFile.Close()
	}

	if lu.dstFile != nil {
		err := lu.writeFile()
		if err != nil {
			return err
		}
		lu.dstFile.Close()
	}
	return nil
}

func (lu *lu) writeFile() error {
	if lu.opts.Sort {
		sort.Sort(entriesByReq(lu.history))
	}

	text := lu.fileTemplater.entry() + lu.fileTemplater.list()
	if lf, ok := lu.fileTemplater.(layoutTemplater); ok {
		text += lf.layout()
	}
	t := template.Must(template.New("").Funcs(templatesFnMap).Parse(text))

	var b bytes.Buffer
	err := t.Execute(&b, struct{ Entries []*entry }{lu.history})
	if err != nil {
		return err
	}

	fmt.Fprint(lu.dstFile, b.String())
	return nil
}
