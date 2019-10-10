#!/usr/bin/env bash

genprotocol -ver=1.0 \
    -basedir=. \
    -prefix=ws
goimports -w ws_packet/packet_gen.go
goimports -w ws_msgp/serialize_gen.go
goimports -w ws_json/serialize_gen.go
goimports -w ws_server/demuxreq2api_gen.go
goimports -w ws_client/recvrspobjfnmap_gen.go
goimports -w ws_client/recvnotiobjfnmap_gen.go
goimports -w ws_client/callsendrecv_gen.go
goimports -w ws_wasmconn/wasmconn_gen.go
goimports -w ws_loopwsgorilla/loopwsgorilla_gen.go
goimports -w ws_looptcp/looptcp_gen.go
