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

// Lu is the main workhorse of the app.
// It holds all the objects needed to perform the job.
type Lu struct {
	dictionary dictionary
	translator translator
	// parsed command line flags
	opts options
	// line by line data source scanner
	scanner *bufio.Scanner
	// templater used to write to output file
	fileTemplater templater
	srcFile       *os.File
	dstFile       *os.File
	// history of all requests and responses
	history []*entry
}

// dictionary defines interface which is used instead of Dictionary struct from yandex-dictionary package
// other implementation is a mock, used for tests and debug
type dictionary interface {
	Lookup(params *yd.Params) (*yd.Entry, error)
}

// translator defines interface which is used instead of Translator struct from yandex-translate package
// other implementation is a mock, used for tests and debug
type translator interface {
	Translate(lang, text string) (*yt.Response, error)
	GetLangs(ui string) (*yt.Languages, error)
}

// entry holds request and corresponding responses, one for each specified language
type entry struct {
	Request   string
	Responses []*response
}

// response holds the single response
type response struct {
	Lang         string
	Translations []string
}

// entriesByReq is the synonym for the entries pointers list, needed for sorting
type entriesByReq []*entry

func (br entriesByReq) Len() int           { return len(br) }
func (br entriesByReq) Swap(i, j int)      { br[i], br[j] = br[j], br[i] }
func (br entriesByReq) Less(i, j int) bool { return br[i].Request < br[j].Request }

// newLu creates the new instance of Lu struct
func newLu(args []string, opts options) (*Lu, error) {
	lu := &Lu{opts: opts}

	err := lu.setup(args)
	if err != nil {
		return nil, err
	}

	return lu, nil
}

// close cleans up the resources allocated by instance of lu
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

// writeFile writes history, possibly sorted, to the specified output file
func (lu *Lu) writeFile() error {
	if lu.opts.Sort {
		sort.Sort(entriesByReq(lu.history))
	}

	text := lu.fileTemplater.entry() + lu.fileTemplater.list()
	// if templater supports layout, use it
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

// setup prepares lu instance
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

// setupAPI sets dictionary and translator, mock ones for tests
// (real tests for dicionary and translator are in corresponding packages)
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

// setupInput sets the data source, it can be stdin, string built from command line arguments or source file
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

// setupFileOutput sets the destination file and templater, if destination file name is specified
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
