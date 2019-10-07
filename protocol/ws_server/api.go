package ws_server

import (
	"github.com/kasworld/wasmwebsocket/protocol/ws_error"
	"github.com/kasworld/wasmwebsocket/protocol/ws_obj"
)

func apifn_ReqInvalidCmd(c2sc *ServeClientConn, robj *ws_obj.ReqInvalidCmd_data) (*ws_obj.RspInvalidCmd_data, error) {
	spacket := &ws_obj.RspInvalidCmd_data{
		Error: ws_error.Disconnect,
	}
	return spacket, nil
}

func apifn_ReqLogin(c2sc *ServeClientConn, robj *ws_obj.ReqLogin_data) (*ws_obj.RspLogin_data, error) {
	spacket := &ws_obj.RspLogin_data{
		Error: ws_error.None,
	}
	return spacket, nil
}

func apifn_ReqHeartbeat(c2sc *ServeClientConn, robj *ws_obj.ReqHeartbeat_data) (*ws_obj.RspHeartbeat_data, error) {
	spacket := &ws_obj.RspHeartbeat_data{
		Error: ws_error.None,
	}
	return spacket, nil
}

func apifn_ReqChat(c2sc *ServeClientConn, robj *ws_obj.ReqChat_data) (*ws_obj.RspChat_data, error) {
	spacket := &ws_obj.RspChat_data{
		Error: ws_error.None,
	}
	return spacket, nil
}
