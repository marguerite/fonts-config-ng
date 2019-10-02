#!/bin/sh

for i in `find . -name "*.go" -type f`; do
  gofmt -s -w ${i};
done
