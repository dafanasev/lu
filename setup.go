package main

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"

	yd "github.com/dafanasev/go-yandex-dictionary"
	yt "github.com/dafanasev/go-yandex-translate"
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

func (lu *lu) setup(args []string) error {
	dictionaryAPIKey := os.Getenv("LU_YANDEX_DICTIONARY_API_KEY")
	if dictionaryAPIKey == "" {
		return errors.New("the required environment variable LU_YANDEX_DICTIONARY_API_KEY is not set")
	}

	translateAPIKey := os.Getenv("LU_YANDEX_TRANSLATE_API_KEY")
	if translateAPIKey == "" {
		return errors.New("the required environment variable LU_YANDEX_TRANSLATE_API_KEY is not set")
	}

	r, err := lu.setupInput(args)
	if err != nil {
		return err
	}
	lu.scanner = bufio.NewScanner(r)

	err = lu.setupFileOutput()
	if err != nil {
		return err
	}

	lu.dictionary = yd.New(dictionaryAPIKey)
	lu.translator = yt.New(translateAPIKey)

	return nil
}

func (lu *lu) setupInput(args []string) (io.Reader, error) {
	if len(args) > 0 {
		req := strings.Join(args, " ")
		return strings.NewReader(req), nil
	}
	if lu.opts.SrcFileName != "" {
		var err error
		lu.srcFile, err = os.Open(lu.opts.SrcFileName)
		if err != nil {
			return nil, err
		}
		return lu.srcFile, nil
	}
	return os.Stdin, nil
}

func (lu *lu) setupFileOutput() error {
	if lu.opts.DstFileName != "" {
		var err error
		lu.dstFile, err = os.OpenFile(lu.opts.DstFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}

		lu.fileTemplater = func(ext string) templater {
			if ext == "html" {
				return &htmlTemplater{}
			}
			return &textTemplater{}
		}(filepath.Ext(lu.opts.DstFileName)[1:])
	}
	return nil
}
