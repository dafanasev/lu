[![Build Status](https://travis-ci.org/dafanasev/lu.svg?branch=master)](https://travis-ci.org/dafanasev/lu)
[![Go Report Card](https://goreportcard.com/badge/github.com/dafanasev/lu)](https://goreportcard.com/report/github.com/dafanasev/lu)
[![Coverage Status](https://coveralls.io/repos/github/dafanasev/lu/badge.svg)](https://coveralls.io/github/dafanasev/lu)

# lu

lu is a terminal client for Yandex.Dictionary and Yandex.Translate services.

In order to use it please set the LU_YANDEX_DICTIONARY_API_KEY and LU_YANDEX_TRANSLATE_API_KEY environment 
variables. The corresponding API keys can be obtained at https://api.yandex.ru

## Features:

* gets stuff to translate from command line arguments, from files (one lookup per file) or interactively from STDIN
* multiple languages to translate to
* outputs translation to STDOUT, text or html files. 
* output can be sorted alphabetically by request strings
* default languages to translate from and to can be specified using environment variables

## Install

You can build lu from source using the `go get -u github.com/dafanasev/lu` command or download binaries from https://github.com/dafanasev/lu/releases   
    

## Usage:
```  
lu [OPTIONS]

Application Options:
  -f, --from=      language to translate from [$LU_DEFAULT_FROM_LANG]
  -t, --to=        languages to translate to [$LU_DEFAULT_TO_LANGS]
  -i, --source=    source file name
  -o, --output=    destination file name
  -s, --sort       sort alphabetically
  -l, --languages  show supported languages
  -v, --version    show version

Help Options:
  -h, --help       Show this help message
```

The `$LU_DEFAULT_TO_LANGS` environment variable can be used to specify a list of destination languages, with the colon used as separator, e.g. `ru:it:de`

## Examples:

`$ lu -fen -tde -i in.txt -o out.txt` 

translates stuff from in.txt from english to german and writes translations to out.txt

`$ lu -i in.txt -o out.html`

translates stuff from in.txt using default languages specified in the $LU_DEFAULT_FROM_LANG and $LU_DEFAULT_TO_LANGS 
environment variables and writes translations to out.html

`$ lu -o out.html -s`

translates stuff from STDIN and writes translations to out.html sorted by requests phrases

`$ lu`
 
translates stuff from STDIN using default languages adn writes translations to STDOUT