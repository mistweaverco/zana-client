#!/usr/bin/env bash

test_coverage() {
  go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out
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
