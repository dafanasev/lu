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

// TODO: use templates to render to terminal
// TODO: add indexes to text output
// TODO: show progress if output only to file
// TODO: html file layout translations

var (
	dictionary    *yandex_dictionary.Dictionary
	translator    *yandex_translate.Translator
	scanner       *bufio.Scanner
	fileFormatter entryFormatter
	srcFile       *os.File
	dstFile       *os.File
	history       []*Entry
)

type Entry struct {
	Request   string
	Responses []*Response
}

type Response struct {
	Lang         string
	Translations []string
}

type byReq []*Entry

func (br byReq) Len() int           { return len(br) }
func (br byReq) Swap(i, j int)      { br[i], br[j] = br[j], br[i] }
func (br byReq) Less(i, j int) bool { return br[i].Request < br[j].Request }

type entryFormatter interface {
	entryTmpl() string
	listTmpl() string
}

var opts struct {
	FromLang    string   `short:"f" env:"LU_DEFAULT_FROM_LANG" required:"true" description:"default language to translate from"`
	ToLangs     []string `short:"t" env:"LU_DEFAULT_TO_LANG" required:"true" description:"default language to translate to"`
	SrcFileName string   `short:"i" description:"source file name"`
	DstFileName string   `short:"o" description:"destination file name"`
	Sort        bool     `short:"s" description:"sort alphabetically"`
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
		req := strings.Join(args, " ")
		r = strings.NewReader(req)
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
	err := setup()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	go handleExitSignal()

	t := template.Must(template.New("").Parse((&textFormatter{}).entryTmpl() + "{{ template \"entry\" . }}\n"))

	for scanner.Scan() {
		req := strings.TrimSpace(scanner.Text())
		if req != "" {
			entry := &Entry{Request: req}

			for _, lang := range opts.ToLangs {
				translations := lookup(req, lang)
				resp := &Response{Lang: lang, Translations: translations}
				entry.Responses = append(entry.Responses, resp)
			}

			// print to stdout if here is no destination file - i.e. destination is stdout
			// or if there is no source file, because in this case source was stdin
			// and we want to see output in the terminal too, even if the destination file is specified
			if srcFile == nil || dstFile == nil {
				t.Execute(os.Stdout, entry)
			}

			history = append(history, entry)
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

func handleExitSignal() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	cleanUp()

	os.Exit(0)
}

func lookup(req string, lang string) []string {
	dictResp, err := dictionary.Lookup(&yandex_dictionary.Params{Lang: opts.FromLang + "-" + lang, Text: req})

	if err == nil {
		var trs []string
		for _, def := range dictResp.Def {
			for _, tr := range def.Tr {
				trs = append(trs, tr.Text)
			}
		}
		return trs
	}

	transResp, err := translator.Translate(lang, req)
	if err != nil || transResp.Result() == req {
		return []string{"no translation"}
	}

	return []string{transResp.Result()}
}

func writeFile() {
	if opts.Sort {
		sort.Sort(byReq(history))
	}

	t := template.Must(template.New("").Parse(fileFormatter.entryTmpl() + fileFormatter.listTmpl()))
	var b bytes.Buffer
	err := t.Execute(&b, struct{ Entries []*Entry }{history})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprint(dstFile, b.String())
}

type textFormatter struct{}

func (f *textFormatter) entryTmpl() string {
	return `
{{- define "entry" }}
{{.Request}}
**********************************************************
{{- range .Responses }}
{{.Lang}}:
{{ range .Translations -}}
{{.}}
{{ end -}}
----------------------------------------------------------
{{- end }}
{{- end }}`
}

func (f *textFormatter) listTmpl() string {
	return `
{{- range .Entries -}}
{{.Request}}
{{ end }}
**********************************************************
{{ range .Entries -}}
{{ template "entry" . }}
{{ end }}
`
}

type htmlFormatter struct{}

func (f *htmlFormatter) entryTmpl() string {
	return `{{- define "entry" }}
	<dt>{{.Request}}</dt>
	{{ range .Responses -}}
	<dd>
		<header>{{.Lang}}</header>
		<ul>
			{{ range .Translations -}}
			<li>{{.}}</li>
			{{ end }}
		</ul>
	</dd>
	{{ end -}}
	{{ end }}`
}

func (f *htmlFormatter) listTmpl() string {
	return `<ul>{{range .Entries}}
	<li>{{.Request}}</li>{{end}}
</ul>

<dl>
	{{- range .Entries }}
	{{ template "entry" . }}
	{{- end }}
</dl>
`
}
