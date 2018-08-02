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
	fileTemplater templater
	srcFile       *os.File
	dstFile       *os.File
	history       []*Entry
)

var version = "0.1.0"

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

func main() {
	err := setup()
	if err != nil {
		exitWithError(err)
	}

	if opts.Version {
		fmt.Println(version)
		os.Exit(0)
	}

	if opts.GetLangs {
		langs, err := supportedLangs()
		if err != nil {
			exitWithError(err)
		}
		for _, lang := range langs {
			fmt.Println(lang)
		}
		os.Exit(0)
	}

	go handleExitSignal()

	funcMap := template.FuncMap{"inc": inc}
	t := template.Must(template.New("").Funcs(funcMap).Parse((&textTemplater{}).entry() + "{{ template \"entry\" . }}\n"))

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

func exitWithError(err error) {
	fmt.Println(err)
	os.Exit(1)
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

	tmpl := fileTemplater.entry() + fileTemplater.list()
	if lf, ok := fileTemplater.(layoutTemplater); ok {
		tmpl += lf.layout()
	}
	funcMap := template.FuncMap{"inc": inc, "dict": dict}
	t := template.Must(template.New("").Funcs(funcMap).Parse(tmpl))

	var b bytes.Buffer
	err := t.Execute(&b, struct{ Entries []*Entry }{history})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprint(dstFile, b.String())
}

func supportedLangs() ([]string, error) {
	resp, err := translator.GetLangs("en")
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
