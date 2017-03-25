#!/usr/bin/env bash
dir=bin/windows
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o $dir/natgo-client.exe natgo-client.go
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o $dir/natgo-server.exe natgo-server.go
cd $dir
tar czvf natgo-windows.tar.gz natgo-client.exe natgo-server.exe