#!/bin/bash
set -e
cd /home/xom3a/Projects/ewalletV3
GOPATH=/tmp/gopath GOMODCACHE=/tmp/gopath/pkg/mod /home/xom3a/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.3.linux-amd64/bin/go build -o ewallet .
./ewallet
