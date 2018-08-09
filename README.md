[![Build Status](https://travis-ci.org/dafanasev/lu.svg?branch=master)](https://travis-ci.org/dafanasev/lu)
[![Go Report Card](https://goreportcard.com/badge/github.com/dafanasev/lu)](https://goreportcard.com/report/github.com/dafanasev/lu)
[![Coverage Status](https://coveralls.io/repos/github/dafanasev/lu/badge.svg)](https://coveralls.io/github/dafanasev/lu)

dt
==

dt is a command line client for Yandex Dictionary and Yandex Translate APIs.

It can 

In order to use it please set the TRANSLATE_YANDEX_DICTIONARY_API_KEY and TRANSLATE_YANDEX_TRANSLATE_API_KEY environment 
variables.



`translate en ru in.txt out.txt`

there parameters are as follows: source language, destination language, source file, destination file.
Source and destination files should be different ones.

Features:
    * gets stuff to translate from command line arguments or from file
    * outputs translation to terminal, text or html files. 
     