package ws_handlereq

import (
	"github.com/kasworld/wasmwebsocket/protocol/ws_error"
	"github.com/kasworld/wasmwebsocket/protocol/ws_obj"
	"github.com/kasworld/wasmwebsocket/protocol/ws_packet"
)

func apifn_ReqInvalidCmd(
	me interface{}, hd ws_packet.Header, robj *ws_obj.ReqInvalidCmd_data) (
	ws_packet.Header, *ws_obj.RspInvalidCmd_data, error) {
	rhd := ws_packet.Header{
		ErrorCode: ws_error.Disconnect,
	}
	spacket := &ws_obj.RspInvalidCmd_data{}
	return rhd, spacket, nil
}

func apifn_ReqLogin(
	me interface{}, hd ws_packet.Header, robj *ws_obj.ReqLogin_data) (
	ws_packet.Header, *ws_obj.RspLogin_data, error) {
	rhd := ws_packet.Header{
		ErrorCode: ws_error.None,
	}
	spacket := &ws_obj.RspLogin_data{}
	return rhd, spacket, nil
}

func apifn_ReqHeartbeat(
	me interface{}, hd ws_packet.Header, robj *ws_obj.ReqHeartbeat_data) (
	ws_packet.Header, *ws_obj.RspHeartbeat_data, error) {
	rhd := ws_packet.Header{
		ErrorCode: ws_error.None,
	}
	spacket := &ws_obj.RspHeartbeat_data{}
	return rhd, spacket, nil
}

func apifn_ReqChat(
	me interface{}, hd ws_packet.Header, robj *ws_obj.ReqChat_data) (
	ws_packet.Header, *ws_obj.RspChat_data, error) {
	rhd := ws_packet.Header{
		ErrorCode: ws_error.None,
	}
	spacket := &ws_obj.RspChat_data{}
	return rhd, spacket, nil
}
