package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/dafanasev/go-yandex-dictionary"
	"github.com/dafanasev/go-yandex-translate"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
)

// TODO: translate to multiple languages

var (
	dictionary    *yandex_dictionary.YandexDictionary
	translator    *yandex_translate.Translator
	scanner       *bufio.Scanner
	fileFormatter entryFormatter
	srcFile       *os.File
	dstFile       *os.File
	cache         []*Entry
)

type Entry struct {
	Req          string
	Translations []string
}

type ByReq []*Entry

func (br ByReq) Len() int           { return len(br) }
func (br ByReq) Swap(i, j int)      { br[i], br[j] = br[j], br[i] }
func (br ByReq) Less(i, j int) bool { return br[i].Req < br[j].Req }

type entryFormatter interface {
	template() string
}

var opts struct {
	FromLang    string `short:"f" env:"LU_DEFAULT_FROM_LANG" required:"true" description:"default language to translate from"`
	ToLang      string `short:"t" env:"LU_DEFAULT_TO_LANG" required:"true" description:"default language to translate to"`
	SrcFileName string `short:"i" description:"source file name"`
	DstFileName string `short:"o" description:"destination file name"`
	Sort        bool   `short:"s" description:"sort alphabetically"`
}

func setup() error {
	dictionaryAPIKey := os.Getenv("LU_YANDEX_DICTIONARY_API_KEY")
	if dictionaryAPIKey == "" {
		return errors.New("the required environment variable LU_YANDEX_DICTIONARY_API_KEY is not set")
	}

	translateAPIKey := os.Getenv("LU_YANDEX_TRANSLATE_API_KEY")
	if translateAPIKey == "" {
		return errors.New("the required environment variable LU_YANDEX_TRANSLATE_API_KEY is not set")
	}

	args, err := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash).Parse()
	if err != nil {
		return errors.Wrap(err, "can not parse arguments")
	}

	err = setupInput(args)
	if err != nil {
		return err
	}

	err = setupOutput()
	if err != nil {
		return err
	}

	dictionary = yandex_dictionary.New(dictionaryAPIKey)
	translator = yandex_translate.New(translateAPIKey)

	return nil
}

func setupInput(args []string) error {
	if opts.SrcFileName != "" && opts.SrcFileName == opts.DstFileName {
		return errors.New("source and destination must be different files")
	}

	var r io.Reader = os.Stdin
	if len(args) > 0 {
		entry := strings.Join(args, " ")
		r = strings.NewReader(entry)
	}
	if opts.SrcFileName != "" {
		var err error
		srcFile, err = os.Open(opts.SrcFileName)
		if err != nil {
			return err
		}
		r = srcFile
	}
	scanner = bufio.NewScanner(r)
	return nil
}

func setupOutput() error {
	if opts.DstFileName != "" {
		var err error
		dstFile, err = os.OpenFile(opts.DstFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}

		if filepath.Ext(opts.DstFileName) == ".html" {
			fileFormatter = &htmlFormatter{}
		} else {
			fileFormatter = &textFormatter{}
		}
	}
	return nil
}

func main() {
	go handleExit()

	err := setup()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	delimiter := strings.Repeat("*", 80) + "\n"
	for scanner.Scan() {
		req := strings.TrimSpace(scanner.Text())
		if req != "" {
			translations := lookup(req)
			// print to stdout if here is no destination file - i.e. destination is stdout
			// or if there is no source file, because in this case source was stdin
			// and we want to see output in the terminal too, even if the destination file is specified
			if srcFile == nil || dstFile == nil {
				for i, t := range translations {
					fmt.Printf("%d. %s\n", i+1, t)
				}
				fmt.Print(delimiter)
			}

			if dstFile != nil {
				cache = append(cache, &Entry{req, translations})
			}
		}
	}

	cleanUp()
}

func cleanUp() {
	if srcFile != nil {
		srcFile.Close()
	}

	if dstFile != nil {
		writeFile()
		dstFile.Close()
	}
}

func handleExit() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	cleanUp()

	os.Exit(0)
}

func lookup(req string) []string {
	dictResp, err := dictionary.Lookup(&yandex_dictionary.Params{Lang: opts.FromLang + "-" + opts.ToLang, Text: req})

	if err != nil {
		fmt.Println(err)
	}
	if err == nil {
		var trs []string
		for _, def := range dictResp.Def {
			for _, tr := range def.Tr {
				trs = append(trs, tr.Text)
			}
		}
		return trs
	}

	transResp, err := translator.Translate(opts.ToLang, req)
	if err != nil {
		fmt.Println(err)
		return []string{"no translation"}
	}

	return []string{transResp.Result()}
}

func writeFile() {
	if opts.Sort {
		sort.Sort(ByReq(cache))
	}

	t := template.Must(template.New("text").Parse(fileFormatter.template()))
	var b bytes.Buffer
	err := t.Execute(&b, struct{ Entries []*Entry }{cache})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprint(dstFile, b.String())
}

type textFormatter struct{}

func (f *textFormatter) template() string {
	return `
{{- range .Entries -}}
{{ .Req}}
{{end -}}
**********************************************************
**********************************************************
{{ range .Entries -}}
{{ .Req}}:
{{range .Translations -}}
{{ .}}
{{end -}}
**********************************************************
{{end}}`
}

type htmlFormatter struct{}

func (f *htmlFormatter) template() string {
	return `<ul>{{range .Entries}}
	<li>{{.Req}}</li>{{end}}
</ul>

<dl>
	{{- range .Entries }}
	<dt>{{.Req}}</dt>
		{{ range .Translations -}}
			<dd>{{.}}</dd>
		{{end }}
	{{- end }}
</dl>
`
}
