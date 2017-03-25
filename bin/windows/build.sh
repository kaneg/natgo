#!/usr/bin/env bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/windows/natgo-client.exe natgo-client.go
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/windows/natgo-server.exe natgo-server.go