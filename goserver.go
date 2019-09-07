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
	"net/http"
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

	ServerPacketReadTimeoutSec  = 6
	ServerPacketWriteTimeoutSec = 3
)

func main() {
	port := flag.String("port", ":8080", "Serve port")
	folder := flag.String("dir", ".", "Serve Dir")
	flag.Parse()
	fmt.Printf("server dir=%v port=%v , http://localhost%v/ \n",
		*folder, *port, *port)

	ctx := context.Background()

	webMux := http.NewServeMux()
	webMux.Handle("/",
		http.FileServer(http.Dir(*folder)),
	)
	webMux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWebSocketClient(ctx, w, r)
	})

	if err := http.ListenAndServe(*port, webMux); err != nil {
		fmt.Println(err.Error())
	}
}

func CheckOrigin(r *http.Request) bool {
	return true
}

func serveWebSocketClient(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	upgrader := websocket.Upgrader{
		CheckOrigin: CheckOrigin,
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		gLog.Error("upgrade %v", err)
		return
	}

	gLog.Debug("Start serveWebSocketClient %v", r.RemoteAddr)
	defer func() {
		gLog.Debug("End serveWebSocketClient %v", r.RemoteAddr)
	}()

	c2sc := NewWebSocketConnection(r.RemoteAddr)
	c2sc.ServeWebSocketConnection(ctx, wsConn)

	// connected user play

	// end play
	wsConn.Close()

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

func (c2sc *WebSocketConnection) ServeWebSocketConnection(mainctx context.Context, wsConn *websocket.Conn) {

	gLog.Debug("Start ServeWebSocketConnection %s", c2sc)
	defer func() { gLog.Debug("End ServeWebSocketConnection %s", c2sc) }()

	sendRecvCtx, sendRecvCancel := context.WithCancel(mainctx)
	c2sc.sendRecvStop = sendRecvCancel

	go func() {
		err := gorillawebsocketsendrecv.RecvLoop(sendRecvCtx, c2sc.sendRecvStop, wsConn,
			ServerPacketReadTimeoutSec, c2sc.HandleRecvPacket)
		if err != nil {
			gLog.Error("end RecvLoop %v", err)
		}
	}()
	go func() {
		err := gorillawebsocketsendrecv.SendLoop(sendRecvCtx, c2sc.sendRecvStop, wsConn,
			ServerPacketWriteTimeoutSec, c2sc.sendCh, c2sc.handleSentPacket)
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
