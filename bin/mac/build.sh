#!/usr/bin/env bash
dir=bin/mac
go build -o $dir/natgo-client natgo-client.go
go build -o $dir/natgo-server natgo-server.go
cd $dir
tar czvf natgo-mac.tar.gz natgo-client natgo-server