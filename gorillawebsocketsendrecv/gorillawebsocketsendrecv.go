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
	"github.com/kasworld/wasmwebsocket/wspacket"
)

func SendControl(
	wsConn *websocket.Conn, mt int, PacketWriteTimeOut time.Duration) error {

	return wsConn.WriteControl(mt, []byte{}, time.Now().Add(PacketWriteTimeOut))
}

func SendPacket(wsConn *websocket.Conn, sendBuffer []byte) error {
	return wsConn.WriteMessage(websocket.BinaryMessage, sendBuffer)
}

func Packet2Bytes(pk *wspacket.Packet, buf []byte,
	marshalBodyFn func(interface{}) ([]byte, error)) (int, error) {
	bodyData, err := marshalBodyFn(pk.Body)
	if err != nil {
		return 0, err
	}
	if wspacket.EnableCompress && len(bodyData) > wspacket.CompressLimit {
		// oldlen := len(bodyData)
		bodyData, err = wspacket.CompressData(bodyData)
		if err != nil {
			return 0, err
		}
		pk.Header.SetFlag(wspacket.HF_Compress)
	}
	bodyLen := len(bodyData)
	if bodyLen > wspacket.MaxBodyLen {
		return bodyLen + wspacket.HeaderLen,
			fmt.Errorf("fail to serialize large packet %v, %v", pk.Header, bodyLen)
	}
	pk.Header.BodyLen = uint32(bodyLen)
	copy(buf, pk.Header.ToBytes())
	copy(buf[wspacket.HeaderLen:], bodyData)
	return bodyLen + wspacket.HeaderLen, nil
}

func SendLoop(sendRecvCtx context.Context, SendRecvStop func(), wsConn *websocket.Conn,
	timeout time.Duration,
	SendCh chan wspacket.Packet,
	marshalBodyFn func(interface{}) ([]byte, error),
	handleSentPacketFn func(header wspacket.Header) error,
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
			sendBuffer := wspacket.NewSendPacketBuffer()
			sendN, err := Packet2Bytes(&pk, sendBuffer, marshalBodyFn)
			if err != nil {
				break loop
			}
			if err = SendPacket(wsConn, sendBuffer[:sendN]); err != nil {
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
	HandleRecvPacketFn func(header wspacket.Header, body []byte) error) error {

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

func RecvPacket(wsConn *websocket.Conn) (wspacket.Header, []byte, error) {
	mt, rdata, err := wsConn.ReadMessage()
	if err != nil {
		return wspacket.Header{}, nil, err
	}
	if mt != websocket.BinaryMessage {
		return wspacket.Header{}, nil, fmt.Errorf("message not binary %v", mt)
	}
	return wspacket.NewRecvPacketBufferByData(rdata).GetHeaderBody()
}
