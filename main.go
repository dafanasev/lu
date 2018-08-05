package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"os"
	"os/signal"
	"sort"
	"syscall"
)

var version = "0.1.0"

var (
	dict          dictionary
	tr            translator
	scanner       *bufio.Scanner
	fileTemplater templater
	srcFile       *os.File
	dstFile       *os.File
	history       []*entry
)

var stdoutTemplater = func() *template.Template {
	return template.Must(template.New("").Funcs(templaesFnMap).Parse((&textTemplater{}).entry() + "{{ template \"entry\" . }}\n"))
}()

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

func main() {
	args, err := parseOpts()
	if err != nil {
		exitWithError(err)
	}

	if opts.Version {
		fmt.Println(version)
		os.Exit(0)
	}

	if opts.ShowLangs {
		err = showLangs()
		if err != nil {
			exitWithError(err)
		}
		os.Exit(0)
	}

	err = setup(args)
	if err != nil {
		exitWithError(err)
	}

	go handleExitSignal()

	entriesChan := make(chan *entry)
	go lookupCycle(entriesChan)

	n := 0
	for entry := range entriesChan {
		n++
		handleEntry(entry)
		printResults(entry, n)
	}

	cleanUp()
}

func handleEntry(entry *entry) {
	history = append(history, entry)
}

func printResults(entry *entry, n int) {
	if shouldPrintEntry() {
		err := stdoutTemplater.Execute(os.Stdout, entry)
		if err != nil {
			exitWithError(err)
		}
	} else {
		fmt.Printf("%d. Got results for %s\n", n, entry.Request)
	}
}

func shouldPrintEntry() bool {
	// print to stdout if there is no destination file - i.e. destination is stdout
	// or if there is no source file, because in this case source is stdin
	// and we want to see output in the terminal too, even if the destination file is specified
	// otherwise show progress
	return srcFile == nil || dstFile == nil
}

func exitWithError(err error) {
	fmt.Println(err)
	os.Exit(1)
}

func showLangs() error {
	langs, err := supportedLangs("en")
	if err != nil {
		return err
	}
	for _, lang := range langs {
		fmt.Println(lang)
	}
	return nil
}

func cleanUp() {
	if srcFile != nil {
		srcFile.Close()
	}

	if dstFile != nil {
		err := writeFile()
		if err != nil {
			exitWithError(err)
		}
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

func writeFile() error {
	if opts.Sort {
		sort.Sort(entriesByReq(history))
	}

	text := fileTemplater.entry() + fileTemplater.list()
	if lf, ok := fileTemplater.(layoutTemplater); ok {
		text += lf.layout()
	}
	t := template.Must(template.New("").Funcs(templaesFnMap).Parse(text))

	var b bytes.Buffer
	err := t.Execute(&b, struct{ Entries []*entry }{history})
	if err != nil {
		return err
	}

	fmt.Fprint(dstFile, b.String())
	return nil
}
