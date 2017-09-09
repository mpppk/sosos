#!/usr/bin/env bash

gox -os "darwin linux windows" \
 -arch "386 amd64" \
 -output "pkg/{{.Dir}}_{{.OS}}_{{.Arch}}/{{.Dir}}"

mkdir tarpkg

tar -zcvf tarpkg/sosos_linux_amd64.tar.gz pkg/sosos_linux_amd64/sosos
tar -zcvf tarpkg/sosos_linux_386.tar.gz pkg/sosos_linux_386/sosos
tar -zcvf tarpkg/sosos_darwin_amd64.tar.gz pkg/sosos_darwin_amd64/sosos
tar -zcvf tarpkg/sosos_darwin_386.tar.gz pkg/sosos_darwin_386/sosos
zip tarpkg/sosos_windows_amd64.zip pkg/sosos_windows_amd64/sosos.exe
zip tarpkg/sosos_windows_386.zip pkg/sosos_windows_386/sosos.exe

rm -f tarpkg/.DS_Store

ghr v0.8.0 tarpkg