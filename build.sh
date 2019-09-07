#!/usr/bin/env bash


echo "gen log"
loggen -leveldatafile serverloglevel.data -packagename serverlog 

echo "build wasm client"
rm www/wasmwebsocket.wasm
GOOS=js GOARCH=wasm go build  -o www/wasmwebsocket.wasm wasmclient.go

echo " run server"
go run server.go -dir www