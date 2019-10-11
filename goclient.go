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
	"time"

	"github.com/kasworld/wasmwebsocket/protocol/ws_connwsgorilla"
	"github.com/kasworld/wasmwebsocket/protocol/ws_idcmd"
	"github.com/kasworld/wasmwebsocket/protocol/ws_json"
	"github.com/kasworld/wasmwebsocket/protocol/ws_obj"
	"github.com/kasworld/wasmwebsocket/protocol/ws_packet"
)

// service const
const (

	// for client
	PacketReadTimeoutSec  = 6 * time.Second
	PacketWriteTimeoutSec = 3 * time.Second
)

func main() {
	addr := flag.String("addr", "localhost:8080", "server addr")
	flag.Parse()
	fmt.Printf("addr %v \n", *addr)

	app := new(App)
	app.c2sc = ws_connwsgorilla.New(
		PacketReadTimeoutSec, PacketWriteTimeoutSec,
		ws_json.MarshalBodyFn,
		app.HandleRecvPacket,
		app.handleSentPacket,
	)
	if err := app.c2sc.ConnectTo(*addr); err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	app.c2sc.EnqueueSendPacket(app.makePacket())
	ctx := context.Background()
	app.c2sc.Run(ctx)
}

type App struct {
	pid  uint32
	c2sc *ws_connwsgorilla.Connection
}

func (app *App) handleSentPacket(header ws_packet.Header) error {
	return nil
}

func (app *App) HandleRecvPacket(header ws_packet.Header, body []byte) error {
	robj, err := ws_json.UnmarshalPacket(header, body)
	_ = robj
	// fmt.Println(header, robj, err)
	if err == nil {
		app.c2sc.EnqueueSendPacket(app.makePacket())
	}
	return err
}

func (app *App) makePacket() ws_packet.Packet {
	body := ws_obj.ReqHeartbeat_data{}
	hd := ws_packet.Header{
		Cmd:      uint16(ws_idcmd.Heartbeat),
		ID:       app.pid,
		FlowType: ws_packet.Request,
	}
	app.pid++

	return ws_packet.Packet{
		Header: hd,
		Body:   body,
	}
}
