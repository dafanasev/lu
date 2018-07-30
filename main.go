package main

import (
	"bufio"
	"fmt"
	"io"
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
// TODO: try to redo using text/html tmplates

var (
	dictionary    *yandex_dictionary.YandexDictionary
	translator    *yandex_translate.Translator
	scanner       *bufio.Scanner
	fileFormatter entryFormatter
	srcFile       *os.File
	dstFile       *os.File
	cache         []*entry
)

type entry struct {
	req          string
	translations []string
}

type ByReq []*entry

func (br ByReq) Len() int           { return len(br) }
func (br ByReq) Swap(i, j int)      { br[i], br[j] = br[j], br[i] }
func (br ByReq) Less(i, j int) bool { return br[i].req < br[j].req }

type entryFormatter interface {
	formatRequest(req string) string
	formatTranslations(translations ...string) string
	formatHeader(req string) string
	delimiter() string
}

var opts struct {
	FromLang    string `short:"f" env:"LU_DEFAULT_FROM_LANG" required:"true" description:"default language to translate from"`
	ToLang      string `short:"t" env:"LU_DEFAULT_TO_LANG" required:"true" description:"default language to translate to"`
	SrcFileName string `short:"s" description:"source file name"`
	DstFileName string `short:"d" description:"destination file name"`
	Sort        bool   `short:"a" description:"sort alphabetically"`
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

	stdinFormatter := &textFormatter{}
	delimiter := stdinFormatter.delimiter()
	for scanner.Scan() {
		req := strings.TrimSpace(scanner.Text())
		if req != "" {
			translations := lookup(req)
			// print to stdout if here is no destination file - i.e. destination is stdout
			// or if there is no source file, because in this case source was stdin
			// and we want to see output in the terminal too, even if the destination file is specified
			if srcFile == nil || dstFile == nil {
				fmt.Print(stdinFormatter.formatTranslations(translations...))
				fmt.Print(delimiter)
			}

			if dstFile != nil {
				cache = append(cache, &entry{req, translations})
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

	// print out header at first
	for _, entry := range cache {
		fmt.Fprint(dstFile, fileFormatter.formatHeader(entry.req))
	}

	fmt.Fprint(dstFile, fileFormatter.delimiter())
	fmt.Fprint(dstFile, fileFormatter.delimiter())

	for _, entry := range cache {
		fmt.Fprint(dstFile, fileFormatter.formatRequest(entry.req))
		fmt.Fprint(dstFile, fileFormatter.formatTranslations(entry.translations...))
		fmt.Fprint(dstFile, fileFormatter.delimiter())
	}
}

type textFormatter struct{}

func (f *textFormatter) formatRequest(req string) string {
	return req + ":\n"
}

func (f *textFormatter) formatTranslations(translations ...string) string {
	b := &strings.Builder{}
	for i, tr := range translations {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, tr))
	}
	return b.String()
}

func (f *textFormatter) formatHeader(entry string) string {
	return entry + "\n"
}

func (f *textFormatter) delimiter() string {
	return strings.Repeat("*", 80) + "\n"
}

type htmlFormatter struct{}

func (f *htmlFormatter) formatRequest(req string) string {
	return fmt.Sprintf("<dt>%s</dt>\n", req)
}

func (f *htmlFormatter) formatTranslations(translations ...string) string {
	b := strings.Builder{}
	for _, t := range translations {
		b.WriteString(fmt.Sprintf("<dd>%s</dd>\n", t))
	}
	return b.String()
}

func (f *htmlFormatter) formatHeader(entry string) string {
	return fmt.Sprintf("<li>%s</li>\n", entry)
}

func (f *htmlFormatter) delimiter() string {
	return "\n"
}
