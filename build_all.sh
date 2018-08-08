#!/usr/bin/env bash

GOOS=darwin GOARCH=amd64 packr build
tar cvzf lu_${VERSION}_macos.tar.gz lu
rm lu

GOOS=linux GOARCH=amd64 packr build
tar cvzf lu_${VERSION}_linux-64bit.tar.gz lu
rm lu

GOOS=linux GOARCH=386 packr build
tar cvzf lu_${VERSION}_linux-32bit.tar.gz lu
rm lu

GOOS=windows GOARCH=amd64 packr build -o lu.exe
zip lu_${VERSION}_windows.zip lu.exe
rm lu.exe