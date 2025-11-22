#!/bin/bash

mkdir -p ./build/tmp
mkdir -p ./build/out

GOOS=windows GOARCH=amd64 go build -ldflags -H=windowsgui -o ./build/tmp/Start.exe
cp -r files/ ./build/tmp/compatibility/

cd ./build/tmp
zip -r ../out/release.zip .
cd -
rm -rf ./build/tmp
