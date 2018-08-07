package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	yd "github.com/dafanasev/go-yandex-dictionary"
	yt "github.com/dafanasev/go-yandex-translate"
	"github.com/pkg/errors"
)

// Lu is the struct that does real job
type Lu struct {
	dictionary
	translator
	opts          options
	scanner       *bufio.Scanner
	fileTemplater templater
	srcFile       *os.File
	dstFile       *os.File
	history       []*entry
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

func newLu(args []string, opts options) (*Lu, error) {
	lu := &Lu{opts: opts}

	err := lu.setup(args)
	if err != nil {
		return nil, err
	}

	return lu, nil
}

func (lu *Lu) close() {
	if lu.srcFile != nil {
		lu.srcFile.Close()
		lu.srcFile = nil
	}

	if lu.dstFile != nil {
		lu.dstFile.Close()
		lu.dstFile = nil
	}
}

func (lu *Lu) shouldPrintEntries() bool {
	// print to stdout if there is no destination file - i.e. destination is stdout
	// or if there is no source file, because in this case source is stdin
	// and we want to see output in the terminal too, even if the destination file is specified
	// otherwise show progress
	return lu.srcFile == nil || lu.dstFile == nil
}

func (lu *Lu) writeFile() error {
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

func (lu *Lu) setup(args []string) error {
	err := lu.setupAPI()
	if err != nil {
		return err
	}

	r, err := lu.setupInput(args)
	if err != nil {
		return err
	}
	lu.scanner = bufio.NewScanner(r)

	err = lu.setupFileOutput()
	return err
}

func (lu *Lu) setupAPI() error {
	if os.Getenv("LU_TEST") == "1" {
		lu.dictionary = &dictionaryMock{}
		lu.translator = &translatorMock{}
		return nil
	}
	dictionaryAPIKey := os.Getenv("LU_YANDEX_DICTIONARY_API_KEY")
	if dictionaryAPIKey == "" {
		return errors.New("the required environment variable LU_YANDEX_DICTIONARY_API_KEY is not set")
	}

	translateAPIKey := os.Getenv("LU_YANDEX_TRANSLATE_API_KEY")
	if translateAPIKey == "" {
		return errors.New("the required environment variable LU_YANDEX_TRANSLATE_API_KEY is not set")
	}
	lu.dictionary = yd.New(dictionaryAPIKey)
	lu.translator = yt.New(translateAPIKey)
	return nil
}

func (lu *Lu) setupInput(args []string) (io.Reader, error) {
	if len(args) > 0 {
		req := strings.Join(args, " ")
		return strings.NewReader(req), nil
	}
	if lu.opts.SrcFileName != "" {
		var err error
		lu.srcFile, err = os.Open(lu.opts.SrcFileName)
		if err != nil {
			return nil, err
		}
		return lu.srcFile, nil
	}
	return os.Stdin, nil
}

func (lu *Lu) setupFileOutput() error {
	if lu.opts.DstFileName != "" {
		var err error
		lu.dstFile, err = os.OpenFile(lu.opts.DstFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}

		lu.fileTemplater = func(ext string) templater {
			if ext == "html" {
				return &htmlTemplater{}
			}
			return &textTemplater{}
		}(filepath.Ext(lu.opts.DstFileName)[1:])
	}
	return nil
}
