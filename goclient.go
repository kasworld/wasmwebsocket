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
	"github.com/kasworld/goguelike2/server/g2packet"
	"github.com/kasworld/log/logflags"
	"github.com/kasworld/wasmwebsocket/golog"
	"github.com/kasworld/wasmwebsocket/gorillawebsocketsendrecv"
	"github.com/kasworld/wasmwebsocket/wspacket"
)

var gLog = golog.New("", logflags.DefaultValue(false), golog.LL_All)

// service const
const (
	ServerSendBufferSize = 10

	// for client
	ClientPacketReadTimeoutSec  = 6
	ClientPacketWriteTimeoutSec = 3
)

func main() {
	serverurl := flag.String("url", "http://localhost:8080", "server url")
	flag.Parse()
	fmt.Printf("serverurl %v \n", *serverurl)

	c2sc := NewWebSocketConnection(*serverurl)
	ctx := context.Background()
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
	u := url.URL{Scheme: "ws", Host: c2sc.RemoteAddr, Path: "/ws"}
	wsConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		gLog.Error("dial: %v", err)
		return
	}

	sendRecvCtx, sendRecvCancel := context.WithCancel(mainctx)
	c2sc.sendRecvStop = sendRecvCancel

	go func() {
		err := gorillawebsocketsendrecv.RecvLoop(sendRecvCtx, c2sc.sendRecvStop, wsConn,
			ClientPacketReadTimeoutSec, c2sc.HandleRecvPacket)
		if err != nil {
			gLog.Error("end RecvLoop %v", err)
		}
	}()
	go func() {
		err := gorillawebsocketsendrecv.SendLoop(sendRecvCtx, c2sc.sendRecvStop, wsConn,
			ClientPacketWriteTimeoutSec, c2sc.sendCh, c2sc.handleSentPacket)
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
