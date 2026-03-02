#!/bin/bash
cd coredns-api/coredns && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags='-extldflags "-static" -s -w' -tags netgo -o build/coredns-with-api-centos coredns/coredns.go