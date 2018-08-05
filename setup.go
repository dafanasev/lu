package main

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"

	yd "github.com/dafanasev/go-yandex-dictionary"
	yt "github.com/dafanasev/go-yandex-translate"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
)

type options struct {
	FromLang    string   `short:"f" long:"from" env:"LU_DEFAULT_FROM_LANG" required:"true" description:"default language to translate from"`
	ToLangs     []string `short:"t" long:"to" env:"LU_DEFAULT_TO_LANGS" required:"true" description:"default language to translate to"`
	SrcFileName string   `short:"i" long:"source" description:"source file name"`
	DstFileName string   `short:"o" long:"output" description:"destination file name"`
	Sort        bool     `short:"s" long:"sort" description:"sort alphabetically"`
	ShowLangs   bool     `short:"l" long:"languages" description:"show supported languages"`
	Version     bool     `short:"v" long:"version" description:"show version"`
}

var opts options

func setup(args []string) error {
	dictionaryAPIKey := os.Getenv("LU_YANDEX_DICTIONARY_API_KEY")
	if dictionaryAPIKey == "" {
		return errors.New("the required environment variable LU_YANDEX_DICTIONARY_API_KEY is not set")
	}

	translateAPIKey := os.Getenv("LU_YANDEX_TRANSLATE_API_KEY")
	if translateAPIKey == "" {
		return errors.New("the required environment variable LU_YANDEX_TRANSLATE_API_KEY is not set")
	}

	r, err := setupInput(args)
	if err != nil {
		return err
	}
	scanner = bufio.NewScanner(r)

	err = setupFileOutput()
	if err != nil {
		return err
	}

	dict = yd.New(dictionaryAPIKey)
	tr = yt.New(translateAPIKey)

	return nil
}

func parseOpts() ([]string, error) {
	args, err := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash).Parse()
	if opts.SrcFileName != "" && opts.SrcFileName == opts.DstFileName {
		return nil, errors.New("source and destination must be different files")
	}
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			return nil, err
		}
		return nil, errors.Wrap(err, "can not parse arguments")
	}

	if strings.Contains(opts.ToLangs[0], ":") {
		opts.ToLangs = strings.Split(opts.ToLangs[0], ":")
	}

	return args, nil
}

func setupInput(args []string) (io.Reader, error) {
	if len(args) > 0 {
		req := strings.Join(args, " ")
		return strings.NewReader(req), nil
	}
	if opts.SrcFileName != "" {
		var err error
		srcFile, err = os.Open(opts.SrcFileName)
		if err != nil {
			return nil, err
		}
		return srcFile, nil
	}
	return os.Stdin, nil
}

func setupFileOutput() error {
	if opts.DstFileName != "" {
		var err error
		dstFile, err = os.OpenFile(opts.DstFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}

		fileTemplater = func(ext string) templater {
			if ext == "html" {
				return &htmlTemplater{}
			}
			return &textTemplater{}
		}(filepath.Ext(opts.DstFileName)[1:])
	}
	return nil
}
