#!/usr/bin/env bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/linux/natgo-client natgo-client.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/linux/natgo-server natgo-server.go