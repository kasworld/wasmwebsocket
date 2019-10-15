#!/usr/bin/env bash


echo "genenerate log"
genlog -leveldatafile gologlevel.data -packagename golog 

cd protocol 
genprotocol -ver=1.0 \
    -basedir=. \
    -prefix=ws

goimports -w ws_version/version_gen.go
goimports -w ws_idcmd/command_gen.go
goimports -w ws_idnoti/noti_gen.go
goimports -w ws_error/error_gen.go
goimports -w ws_packet/packet_gen.go
goimports -w ws_obj/objtemplate_gen.go
goimports -w ws_msgp/serialize_gen.go
goimports -w ws_json/serialize_gen.go
goimports -w ws_handlersp/recvrspobjfnmap_gen.go
goimports -w ws_handlenoti/recvnotiobjfnmap_gen.go
goimports -w ws_callsendrecv/callsendrecv_gen.go
goimports -w ws_handlereq/recvreqobjfnmap_gen.go
goimports -w ws_handlereq/apitemplate_gen.go
goimports -w ws_conntcp/conntcp_gen.go
goimports -w ws_connwasm/connwasm_gen.go
goimports -w ws_connwsgorilla/connwsgorilla_gen.go
goimports -w ws_loopwsgorilla/loopwsgorilla_gen.go
goimports -w ws_looptcp/looptcp_gen.go
goimports -w ws_statnoti/statnoti_gen.go
goimports -w ws_statcallapi/statcallapi_gen.go
goimports -w ws_statserveapi/statserveapi_gen.go
goimports -w ws_statapierror/statapierror_gen.go

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