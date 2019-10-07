#!/usr/bin/env bash


echo "gen log"
loggen -leveldatafile gologlevel.data -packagename golog 

cd protocol 
./gen.sh
cd ..

echo "build wasm client"
rm www/wasmwebsocket.wasm
GOOS=js GOARCH=wasm go build -o www/wasmwebsocket.wasm wasmclient.go

echo "build go client"
go build -o bin/goclient goclient.go


echo "build server"
go build -o bin/goserver goserver.go

# echo " run server"
# go run server.go -dir www