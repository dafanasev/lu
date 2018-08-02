package main

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dafanasev/go-yandex-dictionary"
	"github.com/dafanasev/go-yandex-translate"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
)

var opts struct {
	FromLang    string   `short:"f" long:"from" env:"LU_DEFAULT_FROM_LANG" required:"true" description:"default language to translate from"`
	ToLangs     []string `short:"t" long:"to" env:"LU_DEFAULT_TO_LANGS" required:"true" description:"default language to translate to"`
	SrcFileName string   `short:"i" long:"source" description:"source file name"`
	DstFileName string   `short:"o" long:"output" description:"destination file name"`
	Sort        bool     `short:"s" long:"sort" description:"sort alphabetically"`
	GetLangs    bool     `short:"l" long:"languages" description:"show supported languages"`
	Version     bool     `short:"v" long:"version" description:"show version"`
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
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			return err
		}
		return errors.Wrap(err, "can not parse arguments")
	}

	if strings.Index(opts.ToLangs[0], ":") != -1 {
		opts.ToLangs = strings.Split(opts.ToLangs[0], ":")
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

		fileTemplater = func(ext string) templater {
			if ext == "html" {
				return &htmlTemplater{}
			}
			return &textTemplater{}
		}(filepath.Ext(opts.DstFileName)[1:])
	}
	return nil
}
