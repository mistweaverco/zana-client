#!/usr/bin/env bash

test_coverage() {
	go test -race -covermode=atomic -coverprofile=coverage.out ./...
}

test_unit() {
	go test ./...
}

case $MODE in
  "coverage")
    test_coverage
    ;;
  *)
    test_unit
    ;;
esac
