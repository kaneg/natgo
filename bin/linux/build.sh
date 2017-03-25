#!/usr/bin/env bash
dir=bin/linux
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $dir/natgo-client natgo-client.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $dir/natgo-server natgo-server.go
cd $dir
tar czvf natgo-linux.tar.gz natgo-client natgo-server