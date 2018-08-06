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

var stdoutTemplater = func() *template.Template {
	return template.Must(template.New("").Funcs(templatesFnMap).Parse((&textTemplater{}).entry() + "{{ template \"entry\" . }}\n"))
}()

func main() {
	args, opts, err := parseOpts()
	if err != nil {
		exitWithError(err)
	}

	if opts.Version {
		fmt.Println(version)
		os.Exit(0)
	}

	lu, err := newLu(args, opts)
	if err != nil {
		exitWithError(err)
	}

	if opts.ShowLangs {
		err = lu.showLangs()
		if err != nil {
			exitWithError(err)
		}
		os.Exit(0)
	}

	// TODO: think about contexts
	go handleExitSignal(lu)

	n := 0
	for entry := range lu.ch {
		n++
		printResults(lu, entry, n)
	}

	lu.cleanUp()
}

func parseOpts() ([]string, options, error) {
	var opts options
	args, err := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash).Parse()
	if opts.SrcFileName != "" && opts.SrcFileName == opts.DstFileName {
		return nil, options{}, errors.New("source and destination must be different files")
	}
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			return nil, options{}, err
		}
		return nil, options{}, errors.Wrap(err, "can not parse arguments")
	}

	if strings.Contains(opts.ToLangs[0], ":") {
		opts.ToLangs = strings.Split(opts.ToLangs[0], ":")
	}

	return args, opts, nil
}

func printResults(lu *lu, entry *entry, n int) {
	if lu.shouldPrintEntry() {
		err := stdoutTemplater.Execute(os.Stdout, entry)
		if err != nil {
			exitWithError(err)
		}
	} else {
		fmt.Printf("%d. Got results for %s\n", n, entry.Request)
	}
}

func exitWithError(err error) {
	fmt.Println(err)
	os.Exit(1)
}

func handleExitSignal(lu *lu) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	lu.cleanUp()

	os.Exit(0)
}
