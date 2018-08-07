package main

import (
	"fmt"
	"html/template"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
)

const version = "0.1.0"

type options struct {
	FromLang    string   `short:"f" long:"from" env:"LU_DEFAULT_FROM_LANG" description:"default language to translate from"`
	ToLangs     []string `short:"t" long:"to" env:"LU_DEFAULT_TO_LANGS" description:"default language to translate to"`
	SrcFileName string   `short:"i" long:"source" description:"source file name"`
	DstFileName string   `short:"o" long:"output" description:"destination file name"`
	Sort        bool     `short:"s" long:"sort" description:"sort alphabetically"`
	ShowLangs   bool     `short:"l" long:"languages" description:"show supported languages"`
	Version     bool     `short:"v" long:"version" description:"show version"`
}

var stdoutTemplater = func() *template.Template {
	return template.Must(template.New("").Funcs(templatesFnMap).Parse((&textTemplater{}).entry() + "{{ template \"entry\" . }}\n"))
}()

func main() {
	args, opts, err := parseCommandLine()
	if err != nil {
		exitWithError(err)
	}

	if opts.Version {
		fmt.Printf("lu %s", version)
		return
	}

	lu, err := newLu(args, opts)
	if err != nil {
		exitWithError(err)
	}

	if opts.ShowLangs {
		err = showLangs(lu)
		if err != nil {
			exitWithError(err)
		}
		return
	}

	done := make(chan struct{})
	go handleExitSignal(done)

	ch := make(chan *entry)
	go lu.lookupCycle(done, ch)

	n := 0
	for entry := range ch {
		n++
		printResults(lu, entry, n)
	}

	if lu.dstFile != nil {
		err = lu.writeFile()
		if err != nil {
			exitWithError(err)
		}
	}

	lu.close()
}

func parseCommandLine() ([]string, options, error) {
	var opts options
	args, err := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash).Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			return nil, options{}, err
		}
		return nil, options{}, errors.Wrap(err, "can not parse arguments")
	}

	if opts.SrcFileName != "" && opts.SrcFileName == opts.DstFileName {
		return nil, options{}, errors.New("source and destination must be different files")
	}

	if (opts.FromLang == "" || len(opts.ToLangs) == 0) && !opts.Version && !opts.ShowLangs {
		return nil, options{}, errors.New("translation direction (-f and -t flags must be specified")
	}

	if len(opts.ToLangs) > 0 && strings.Contains(opts.ToLangs[0], ":") {
		opts.ToLangs = strings.Split(opts.ToLangs[0], ":")
	}

	return args, opts, nil
}

func printResults(lu *Lu, entry *entry, n int) {
	if lu.shouldPrintEntries() {
		err := stdoutTemplater.Execute(os.Stdout, entry)
		if err != nil {
			exitWithError(errors.Wrap(err, "can't parse template"))
		}
	} else {
		fmt.Printf("%d. Got results for %s\n", n, entry.Request)
	}
}

func exitWithError(err error) {
	fmt.Println(err)
	os.Exit(1)
}

func handleExitSignal(done chan struct{}) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	close(done)
}

func showLangs(lu *Lu) error {
	fmt.Println("Supported languages:")
	langs, err := lu.supportedLangs("en")
	if err != nil {
		return err
	}
	for _, lang := range langs {
		fmt.Println(lang)
	}
	return nil
}
