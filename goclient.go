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
	"flag"
	"fmt"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kasworld/wasmwebsocket/golog"
	"github.com/kasworld/wasmwebsocket/protocol/ws_idcmd"
	"github.com/kasworld/wasmwebsocket/protocol/ws_json"
	"github.com/kasworld/wasmwebsocket/protocol/ws_obj"
	"github.com/kasworld/wasmwebsocket/protocol/ws_packet"
	"github.com/kasworld/wasmwebsocket/protocol/ws_wsgorilla"
)

// service const
const (
	SendBufferSize = 10

	// for client
	PacketReadTimeoutSec  = 6 * time.Second
	PacketWriteTimeoutSec = 3 * time.Second
)

func main() {
	serverurl := flag.String("url", "localhost:8080", "server url")
	flag.Parse()
	fmt.Printf("serverurl %v \n", *serverurl)

	ctx := context.Background()
	c2sc := NewWebSocketConnection(*serverurl)

	c2sc.enqueueSendPacket(c2sc.makePacket())
	c2sc.enqueueSendPacket(c2sc.makePacket())
	c2sc.ConnectWebSocket(ctx)
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
	sendCh       chan ws_packet.Packet
	sendRecvStop func()
	pid          uint32
}

func (c2sc *WebSocketConnection) makePacket() ws_packet.Packet {
	body := ws_obj.ReqHeartbeat_data{}
	hd := ws_packet.Header{
		Cmd:      uint16(ws_idcmd.Heartbeat),
		ID:       c2sc.pid,
		FlowType: ws_packet.Request,
	}
	c2sc.pid++

	return ws_packet.Packet{
		Header: hd,
		Body:   body,
	}
}

func NewWebSocketConnection(remoteAddr string) *WebSocketConnection {
	c2sc := &WebSocketConnection{
		RemoteAddr: remoteAddr,
		sendCh:     make(chan ws_packet.Packet, SendBufferSize),
	}

	c2sc.sendRecvStop = func() {
		golog.GlobalLogger.Fatal("Too early sendRecvStop call %v", c2sc)
	}
	return c2sc
}

func (c2sc *WebSocketConnection) ConnectWebSocket(mainctx context.Context) {

	golog.GlobalLogger.Debug("Start ConnectWebSocket %s", c2sc)
	defer func() { golog.GlobalLogger.Debug("End ConnectWebSocket %s", c2sc) }()

	// connect
	u := url.URL{Scheme: "ws", Host: c2sc.RemoteAddr, Path: "/ws"}
	wsConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		golog.GlobalLogger.Error("dial: %v", err)
		return
	}

	sendRecvCtx, sendRecvCancel := context.WithCancel(mainctx)
	c2sc.sendRecvStop = sendRecvCancel

	go func() {
		err := ws_wsgorilla.RecvLoop(sendRecvCtx, c2sc.sendRecvStop, wsConn,
			PacketReadTimeoutSec, c2sc.HandleRecvPacket)
		if err != nil {
			golog.GlobalLogger.Error("end RecvLoop %v", err)
		}
	}()
	go func() {
		err := ws_wsgorilla.SendLoop(sendRecvCtx, c2sc.sendRecvStop, wsConn,
			PacketWriteTimeoutSec, c2sc.sendCh,
			ws_json.MarshalBodyFn, c2sc.handleSentPacket)
		if err != nil {
			golog.GlobalLogger.Error("end SendLoop %v", err)
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

func (c2sc *WebSocketConnection) handleSentPacket(header ws_packet.Header) error {
	return nil
}

func (c2sc *WebSocketConnection) HandleRecvPacket(header ws_packet.Header, body []byte) error {
	robj, err := ws_json.UnmarshalPacket(header, body)
	_ = robj
	// fmt.Println(header, robj, err)
	if err == nil {
		c2sc.enqueueSendPacket(c2sc.makePacket())
	}
	return err
}

func (c2sc *WebSocketConnection) enqueueSendPacket(pk ws_packet.Packet) error {
	trycount := 10
	for trycount > 0 {
		select {
		case c2sc.sendCh <- pk:
			return nil
		default:
			trycount--
		}
		golog.GlobalLogger.Warn(
			"Send delayed, %s send channel busy %v, retry %v",
			c2sc, len(c2sc.sendCh), 10-trycount)
		time.Sleep(1 * time.Millisecond)
	}

	return fmt.Errorf("Send channel full %v", c2sc)
}
