#!/usr/bin/env bash
dir=bin/linux-arm
GOOS=linux GOARCH=arm CGO_ENABLED=0 go build -o $dir/natgo-client natgo-client.go
GOOS=linux GOARCH=arm CGO_ENABLED=0 go build -o $dir/natgo-server natgo-server.go
cd $dir
tar czvf natgo-linux-arm.tar.gz natgo-client natgo-server