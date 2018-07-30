package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dafanasev/go-yandex-dictionary"
	"github.com/dafanasev/go-yandex-translate"
)

var delimiter = strings.Repeat("*", 80)

func main() {
	if len(os.Args) < 5 {
		exit("Please provide source and destination languages and file names")
	}

	srcLang := os.Args[1]
	dstLang := os.Args[2]

	srcName := os.Args[3]
	dstName := os.Args[4]

	if srcName == dstName {
		exit("Source and destination must be different files")
	}

	dictApiKey := os.Getenv("TRANSLATE_YANDEX_DICTIONARY_API_KEY")
	if dictApiKey == "" {
		exit("Please set the TRANSLATE_YANDEX_DICTIONARY_API_KEY environment variable")
	}
	translateApiKey := os.Getenv("TRANSLATE_YANDEX_TRANSLATE_API_KEY ")
	if translateApiKey == "" {
		exit("Please set the TRANSLATE_YANDEX_TRANSLATE_API_KEY environment variable")
	}

	dictionary := yandex_dictionary.New(dictApiKey)
	translator := yandex_translate.New(translateApiKey)

	src, err := os.Open(srcName)
	exitOnErr(err)
	defer src.Close()

	dst, err := os.OpenFile(dstName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	exitOnErr(err)
	defer dst.Close()

	fmt.Fprintf(dst, time.Now().Format("2006.01.02 15:04:05")+"\n")

	fmt.Fprintln(dst, delimiter)
	fmt.Fprintln(dst, delimiter)

	nProcessed := 0
	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		req := strings.TrimSpace(scanner.Text())

		if req != "" {
			nProcessed++
			processedMsg := fmt.Sprintf("%d. got results for %s\n", nProcessed, req)
			fmt.Fprintf(dst, "%s: \n", req)

			resp, err := dictionary.Lookup(&yandex_dictionary.Params{Lang: fmt.Sprintf("%s-%s", srcLang, dstLang), Text: req})
			if err == nil {
				fmt.Print(processedMsg)
				n := 0
				for _, def := range resp.Def {
					for _, tr := range def.Tr {
						n++
						fmt.Fprintf(dst, "%d. %s\n", n, tr.Text)
					}
				}
			} else {
				translation, err := translator.Translate(dstLang, req)
				resp := translation.Result()
				fmt.Print(processedMsg)

				if err == nil && resp != req {
					fmt.Fprintln(dst, resp)
				} else {
					fmt.Fprintln(dst, "no translation")
				}
			}
			fmt.Fprintln(dst, delimiter)
		}
	}

	fmt.Fprintln(dst, delimiter)
}

func exitOnErr(err error) {
	if err != nil {
		exit(err.Error())
	}
}

func exit(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}
