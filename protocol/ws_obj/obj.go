package ws_obj

import "github.com/kasworld/wasmwebsocket/protocol/ws_error"

type ReqInvalidCmd_data struct {
	Dummy uint8
}
type RspInvalidCmd_data struct {
	Error ws_error.ErrorCode
}

type ReqLogin_data struct {
	Dummy uint8
}
type RspLogin_data struct {
	Error ws_error.ErrorCode
}

type ReqHeartbeat_data struct {
	Dummy uint8
}
type RspHeartbeat_data struct {
	Error ws_error.ErrorCode
}

type ReqChat_data struct {
	Dummy uint8
}
type RspChat_data struct {
	Error ws_error.ErrorCode
}

type NotiBroadcast_data struct {
	Dummy uint8
}
