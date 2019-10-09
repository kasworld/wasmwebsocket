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
	"fmt"
	"sync"
	"syscall/js"
	"time"

	"github.com/kasworld/goguelike2/webclient/jslog"
	"github.com/kasworld/wasmwebsocket/protocol/ws_idcmd"
	"github.com/kasworld/wasmwebsocket/protocol/ws_json"
	"github.com/kasworld/wasmwebsocket/protocol/ws_obj"
	"github.com/kasworld/wasmwebsocket/protocol/ws_packet"
	"github.com/kasworld/wasmwebsocket/protocol/ws_wasmconn"
)

var done chan struct{}

func main() {
	InitApp()
	<-done
}

type App struct {
	wsc      *ws_wasmconn.Connection
	lasttime time.Time
	pid      uint32
}

func InitApp() {
	dst := "ws://localhost:8080/ws"
	app := App{}
	app.wsc = ws_wasmconn.New(dst, ws_json.MarshalBodyFn, handleRecvPacket, handleSentPacket)

	var err error
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err = app.wsc.Connect(ctx, &wg)
	}()
	wg.Wait()
	if err != nil {
		jslog.Errorf("ws_wasmconn.Connect err %v", err)
		return
	}

	fmt.Printf("%v %v", dst, err)
	js.Global().Call("requestAnimationFrame", js.FuncOf(app.jsFrame))
	app.displayFrame()
}

func (app *App) jsFrame(js.Value, []js.Value) interface{} {
	app.displayFrame()
	js.Global().Call("requestAnimationFrame", js.FuncOf(app.jsFrame))
	return nil
}

func (app *App) displayFrame() {
	thistime := time.Now()
	if app.lasttime.Second() == thistime.Second() {
		return
	}
	app.lasttime = thistime
	fmt.Println(thistime)
	err := app.wsc.EnqueueSendPacket(app.makePacket())
	if err != nil {
		done <- struct{}{}
	}
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

func handleRecvPacket(header ws_packet.Header, body []byte) error {
	robj, err := ws_json.UnmarshalPacket(header, body)
	fmt.Println(header, robj, err)
	return err
}

func handleSentPacket(header ws_packet.Header) error {
	return nil
}
