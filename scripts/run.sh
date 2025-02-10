#!/usr/bin/env bash

go run -ldflags "-X 'github.com/mistweaverco/zana-client/cmd/zana.VERSION=${VERSION}'" main.go
