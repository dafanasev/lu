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

// version holds lu current semantic version number
const version = "0.1.0"

// options used by go-flags package to parse command line arguments into.
// For FromLang and ToLangs it can also get values from environment variables
type options struct {
	FromLang    string   `short:"f" long:"from" env:"LU_DEFAULT_FROM_LANG" description:"default language to translate from"`
	ToLangs     []string `short:"t" long:"to" env:"LU_DEFAULT_TO_LANGS" description:"default language to translate to"`
	SrcFileName string   `short:"i" long:"source" description:"source file name"`
	DstFileName string   `short:"o" long:"output" description:"destination file name"`
	Sort        bool     `short:"s" long:"sort" description:"sort alphabetically"`
	ShowLangs   bool     `short:"l" long:"languages" description:"show supported languages"`
	Version     bool     `short:"v" long:"version" description:"show version"`
}

// stdoutTemplater is the compiled entry template used to print results to stdout
var stdoutTemplater = func() *template.Template {
	return template.Must(template.New("").Funcs(templatesFnMap).Parse((&textTemplater{}).entry() + "{{ template \"entry\" . }}\n"))
}()

func main() {
	args, opts, err := parseCommandLine()
	if err != nil {
		exitWithError(err)
	}

	defer func() { fmt.Println("Powered by Yandex.dictionary and Yandex.translate (https://translate.yandex.ru)") }()

	// if -v or -l flags specified, do corresponding action and exit
	if opts.Version {
		fmt.Printf("lu %s\n", version)
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

	// otherwise start lookup cycle
	entriesCh := make(chan *entry)
	go lu.lookupCycle(done, entriesCh)

	// and print out results (or progress, if input AND output file is specified)
	// (see lu.shouldPrintResults method)
	n := 0
	for entry := range entriesCh {
		n++
		printResults(lu, entry, n)
	}

	// when entries channel is closed and destination file is specified write history to it
	if lu.dstFile != nil {
		err = lu.writeFile()
		if err != nil {
			exitWithError(err)
		}
	}

	// and free resources (close files atm)
	lu.close()
}

// parseCommandLine parses command line arguments into lookup argument and application flags
func parseCommandLine() ([]string, options, error) {
	var opts options
	// need new parser because default one has the PrintErrors flag set but we don't need it
	args, err := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash).Parse()
	if err != nil {
		// check if error is actually not an error but the help flag
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			return nil, options{}, err
		}
		return nil, options{}, errors.Wrap(err, "can not parse arguments")
	}

	if opts.SrcFileName != "" && opts.SrcFileName == opts.DstFileName {
		return nil, options{}, errors.New("source and destination must be different files")
	}

	// to and from languages should be specified if we do real work
	if (opts.FromLang == "" || len(opts.ToLangs) == 0) && !opts.Version && !opts.ShowLangs {
		return nil, options{}, errors.New("translation direction (-f and -t flags must be specified")
	}

	// in the environment variable list of destination languages can be specified as a colon separated string
	if len(opts.ToLangs) > 0 && strings.Contains(opts.ToLangs[0], ":") {
		opts.ToLangs = strings.Split(opts.ToLangs[0], ":")
	}

	return args, opts, nil
}

// printResults prints lookup results or progress, depending on srcFile and dstFile values.
// It prints to stdout if there is no destination file - i.e. destination is stdout
// if there is no destination file - i.e. destination is stdout
// or if there is no source file, because in this case source is stdin
// and we want to see output in the terminal too, even if the destination file is specified
// otherwise show progress
func printResults(lu *Lu, entry *entry, n int) {
	if lu.srcFile == nil || lu.dstFile == nil {
		err := stdoutTemplater.Execute(os.Stdout, entry)
		if err != nil {
			exitWithError(errors.Wrap(err, "can't parse template"))
		}
	} else {
		fmt.Printf("%d. Got results for %s\n", n, entry.Request)
	}
}

// exitWithError prints an error to the terminal and terminates app with error
func exitWithError(err error) {
	fmt.Println(err)
	os.Exit(1)
}

// handleExitSignal handles termination signals and gracefully shutdowns the app,
// stopping the work and writing results
func handleExitSignal(done chan struct{}) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	close(done)
}

// showLangs prints supported languages list to the terminal
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
