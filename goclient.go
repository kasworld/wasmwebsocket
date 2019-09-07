// +build ignore

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

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kasworld/goguelike2/server/g2packet"
	"github.com/kasworld/log/logflags"
	"github.com/kasworld/wasmwebsocket/golog"
	"github.com/kasworld/wasmwebsocket/wspacket"
)

var gLog = golog.New("", logflags.DefaultValue(false), golog.LL_All)

// service const
const (
	ServerSendBufferSize        = 10
	ServerPacketReadTimeOutSec  = 6
	ServerPacketWriteTimeoutSec = 3

	// for client
	ClientReadTimeoutSec  = 6
	ClientWriteTimeoutSec = 3
)

func main() {
	serverurl := flag.String("url", "http://localhost:8080", "server url")
	flag.Parse()
	fmt.Printf("serverurl %v \n", *serverurl)

	c2sc := NewWebSocketConnection(*serverurl)
	ctx := context.Background()

}

///////////////////

func (c2sc *WebSocketConnection) String() string {
	return fmt.Sprintf(
		"WebSocketConnection[%v SendCh:%v]",
		c2sc.RemoteAddr,
		len(c2sc.sendCh),
	)
}

type WebSocketConnection struct {
	RemoteAddr   string
	sendCh       chan wspacket.Packet
	sendRecvStop func()
}

func NewWebSocketConnection(remoteAddr string) *WebSocketConnection {
	c2sc := &WebSocketConnection{
		RemoteAddr: remoteAddr,
		sendCh:     make(chan wspacket.Packet, ServerSendBufferSize),
	}

	c2sc.sendRecvStop = func() {
		gLog.Fatal("Too early sendRecvStop call %v", c2sc)
	}
	return c2sc
}

func (c2sc *WebSocketConnection) ConnectWebSocket(mainctx context.Context) {

	gLog.Debug("Start ConnectWebSocket %s", c2sc)
	defer func() { gLog.Debug("End ConnectWebSocket %s", c2sc) }()

	// connect
	u := url.URL{Scheme: "ws", Host: connAddr, Path: "/ws"}
	wsConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		gLog.Error("dial: %v", err)
		return
	}

	sendRecvCtx, sendRecvCancel := context.WithCancel(mainctx)
	c2sc.sendRecvStop = sendRecvCancel

	go func() {
		err := RecvLoop(sendRecvCtx, c2sc.sendRecvStop, wsConn, c2sc.HandleRecvPacket)
		if err != nil {
			gLog.Error("end RecvLoop %v", err)
		}
	}()
	go func() {
		err := SendLoop(sendRecvCtx, c2sc.sendRecvStop, wsConn, c2sc.sendCh, c2sc.handleSentPacket)
		if err != nil {
			gLog.Error("end SendLoop %v", err)
		}
	}()

loop:
	for {
		select {
		case <-sendRecvCtx.Done():
			break loop

		}
	}
}

func (c2sc *WebSocketConnection) handleSentPacket(header wspacket.Header) error {
	return nil
}

func (c2sc *WebSocketConnection) HandleRecvPacket(header wspacket.Header, rbody []byte) error {

	gLog.Debug("Start HandleRecvPacket %v %v", c2sc, header)
	defer func() {
		gLog.Debug("End HandleRecvPacket %v %v", c2sc, header)
	}()

	switch header.PType {
	default:
		gLog.Panic("invalid packet type %s %v", c2sc, header)

	case g2packet.PT_Request:
		switch header.Cmd {
		default:
			gLog.Panic("invalid packet type %s %v", c2sc, header)
		}
	case g2packet.PT_Response:
		switch header.Cmd {
		default:
			gLog.Panic("invalid packet type %s %v", c2sc, header)
		}

	case g2packet.PT_Notification:
		switch header.Cmd {
		default:
			gLog.Panic("invalid packet type %s %v", c2sc, header)
		}
	}

	return nil
}

func (c2sc *WebSocketConnection) enqueueSendPacket(pk wspacket.Packet) error {
	trycount := 10
	for trycount > 0 {
		select {
		case c2sc.sendCh <- pk:
			return nil
		default:
			trycount--
		}
		gLog.Warn(
			"Send delayed, %s send channel busy %v, retry %v",
			c2sc, len(c2sc.sendCh), 10-trycount)
		time.Sleep(1 * time.Millisecond)
	}

	return fmt.Errorf("Send channel full %v", c2sc)
}

////////////////////

func SendControl(
	wsConn *websocket.Conn, mt int, PacketWriteTimeOut time.Duration) error {

	return wsConn.WriteControl(mt, []byte{}, time.Now().Add(PacketWriteTimeOut))
}

func SendPacket(wsConn *websocket.Conn, sendPk *wspacket.Packet) error {
	sendBuffer := wspacket.NewSendPacketBuffer()
	sendN, err := PacketObj2ByteList(sendPk, sendBuffer)
	if err != nil {
		return err
	}
	return wsConn.WriteMessage(websocket.BinaryMessage, sendBuffer[:sendN])
}

func PacketObj2ByteList(pk *wspacket.Packet, buf []byte) (int, error) {
	totalLen := 0
	bodyData, err := json.Marshal(pk.Body)
	// bodyData, err := pk.Body.(msgp.Marshaler).MarshalMsg(buf[wspacket.HeaderLen:wspacket.HeaderLen])
	if err != nil {
		return totalLen, err
	}
	if wspacket.EnableCompress && len(bodyData) > wspacket.CompressLimit {
		// oldlen := len(bodyData)
		bodyData, err = wspacket.CompressData(bodyData)
		if err != nil {
			return totalLen, err
		}
		pk.Header.SetFlag(wspacket.HF_Compress)
		copy(buf[wspacket.HeaderLen:], bodyData)
	}
	bodyLen := len(bodyData)
	if bodyLen > wspacket.MaxBodyLen {
		return bodyLen + wspacket.HeaderLen,
			fmt.Errorf("fail to serialize large packet %v, %v", pk.Header, bodyLen)
	}
	pk.Header.BodyLen = uint32(bodyLen)
	copy(buf, pk.Header.ToBytes())
	// copy(buf[HeaderLen:], bodyData)
	return bodyLen + wspacket.HeaderLen, nil
}

func SendLoop(sendRecvCtx context.Context, SendRecvStop func(), wsConn *websocket.Conn,
	SendCh chan wspacket.Packet,
	handleSentPacket func(header wspacket.Header) error,
) error {

	defer SendRecvStop()
	var err error
loop:
	for {
		select {
		case <-sendRecvCtx.Done():
			err = SendControl(wsConn, websocket.CloseMessage, ServerPacketWriteTimeoutSec)
			break loop
		case pk := <-SendCh:
			if err = wsConn.SetWriteDeadline(time.Now().Add(ServerPacketWriteTimeoutSec)); err != nil {
				break loop
			}
			if err = SendPacket(wsConn, &pk); err != nil {
				break loop
			}
			if err = handleSentPacket(pk.Header); err != nil {
				break loop
			}
		}
	}
	return err
}

func RecvLoop(sendRecvCtx context.Context, SendRecvStop func(), wsConn *websocket.Conn,
	HandleRecvPacket func(header wspacket.Header, body []byte) error) error {

	defer SendRecvStop()
	var err error
loop:
	for {
		select {
		case <-sendRecvCtx.Done():
			break loop
		default:
			if err = wsConn.SetReadDeadline(time.Now().Add(ServerPacketReadTimeOutSec)); err != nil {
				break loop
			}
			if header, body, lerr := RecvPacket(wsConn); lerr != nil {
				if operr, ok := lerr.(*net.OpError); ok && operr.Timeout() {
					continue
				}
				err = lerr
				break loop
			} else {
				if err = HandleRecvPacket(header, body); err != nil {
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
