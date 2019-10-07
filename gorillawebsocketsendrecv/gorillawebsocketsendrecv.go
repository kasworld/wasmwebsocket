// Copyright 2015,2016,2017,2018,2019 SeukWon Kang (kasworld@gmail.com)
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gorillawebsocketsendrecv

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kasworld/bufferpool"
	"github.com/kasworld/wasmwebsocket/protocol/ws_packet"
)

var pBufferPool = bufferpool.New("PacketBufferPool", ws_packet.MaxPacketLen, 100)

func SendControl(
	wsConn *websocket.Conn, mt int, PacketWriteTimeOut time.Duration) error {

	return wsConn.WriteControl(mt, []byte{}, time.Now().Add(PacketWriteTimeOut))
}

func SendPacket(wsConn *websocket.Conn, sendBuffer []byte) error {
	return wsConn.WriteMessage(websocket.BinaryMessage, sendBuffer)
}

func SendLoop(sendRecvCtx context.Context, SendRecvStop func(), wsConn *websocket.Conn,
	timeout time.Duration,
	SendCh chan ws_packet.Packet,
	marshalBodyFn func(body interface{}, oldBuffToAppend []byte) ([]byte, byte, error),
	handleSentPacketFn func(header ws_packet.Header) error,
) error {

	defer SendRecvStop()
	var err error
loop:
	for {
		select {
		case <-sendRecvCtx.Done():
			err = SendControl(wsConn, websocket.CloseMessage, timeout)
			break loop
		case pk := <-SendCh:
			if err = wsConn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
				break loop
			}
			var sendBuffer []byte
			sendBuffer, err := ws_packet.Packet2Bytes(&pk, marshalBodyFn)
			if err != nil {
				break loop
			}
			if err = SendPacket(wsConn, sendBuffer); err != nil {
				break loop
			}
			if err = handleSentPacketFn(pk.Header); err != nil {
				break loop
			}
		}
	}
	return err
}

func RecvLoop(sendRecvCtx context.Context, SendRecvStop func(), wsConn *websocket.Conn,
	timeout time.Duration,
	HandleRecvPacketFn func(header ws_packet.Header, body []byte) error) error {

	defer SendRecvStop()
	var err error
loop:
	for {
		select {
		case <-sendRecvCtx.Done():
			break loop
		default:
			if err = wsConn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
				break loop
			}
			if header, body, lerr := RecvPacket(wsConn); lerr != nil {
				if operr, ok := lerr.(*net.OpError); ok && operr.Timeout() {
					continue
				}
				err = lerr
				break loop
			} else {
				if err = HandleRecvPacketFn(header, body); err != nil {
					break loop
				}
			}
		}
	}
	return err
}

func RecvPacket(wsConn *websocket.Conn) (ws_packet.Header, []byte, error) {
	mt, rdata, err := wsConn.ReadMessage()
	if err != nil {
		return ws_packet.Header{}, nil, err
	}
	if mt != websocket.BinaryMessage {
		return ws_packet.Header{}, nil, fmt.Errorf("message not binary %v", mt)
	}
	return ws_packet.NewRecvPacketBufferByData(rdata).GetHeaderBody()
}
