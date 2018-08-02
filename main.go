package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/dafanasev/go-yandex-dictionary"
	"github.com/dafanasev/go-yandex-translate"
)

// TODO: improve html layout

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

var opts struct {
	FromLang    string   `short:"f" env:"LU_DEFAULT_FROM_LANG" required:"true" description:"default language to translate from"`
	ToLangs     []string `short:"t" env:"LU_DEFAULT_TO_LANG" required:"true" description:"default language to translate to"`
	SrcFileName string   `short:"i" description:"source file name"`
	DstFileName string   `short:"o" description:"destination file name"`
	Sort        bool     `short:"s" description:"sort alphabetically"`
}

func main() {
	err := setup()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	go handleExitSignal()

	funcMap := template.FuncMap{"inc": inc}
	t := template.Must(template.New("").Funcs(funcMap).Parse((&textFormatter{}).entryTmpl() + "{{ template \"entry\" }}\n"))

	n := 0
	for scanner.Scan() {
		req := strings.TrimSpace(scanner.Text())
		if req != "" {
			n++
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
			} else {
				// otherwise show progress
				fmt.Printf("%d. Got results for %s\n", n, req)
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

	tmpl := fileFormatter.entryTmpl() + fileFormatter.listTmpl()
	funcMap := template.FuncMap{"inc": inc, "dict": dict}
	if lf, ok := fileFormatter.(layoutFormatter); ok {
		tmpl += lf.layoutTmpl()
	}
	t := template.Must(template.New("").Funcs(funcMap).Parse(tmpl))

	var b bytes.Buffer
	err := t.Execute(&b, struct{ Entries []*Entry }{history})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprint(dstFile, b.String())
}
