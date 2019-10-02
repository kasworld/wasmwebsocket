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
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/kasworld/wasmwebsocket/logdur"
	"github.com/kasworld/wasmwebsocket/wasmwsconnection"
	"github.com/kasworld/wasmwebsocket/wspacket"
)

var done chan struct{}

func main() {
	InitApp()
	<-done
}

type App struct {
	wsc      *wasmwsconnection.Connection
	lasttime time.Time
	pid      uint32
}

func InitApp() {
	dst := "ws://localhost:8080/ws"
	app := App{}
	app.wsc = wasmwsconnection.New(dst, marshalBodyFn, handleRecvPacket, handleSentPacket)
	err := app.wsc.Connect()
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

func (app *App) makePacket() wspacket.Packet {
	body := "hello world!!"
	hd := wspacket.Header{
		Cmd:   1,
		PkID:  app.pid,
		PType: wspacket.Request,
	}
	app.pid++

	return wspacket.Packet{
		Header: hd,
		Body:   body,
	}
}

func marshalBodyFn(body interface{}) ([]byte, error) {
	return json.Marshal(body)
}

func handleRecvPacket(header wspacket.Header, body []byte) error {
	defer fmt.Printf("end %v\n", logdur.New("handleRecvPacket"))

	fmt.Println(header, body)
	var err error
	switch header.PType {
	default:
		err = fmt.Errorf("invalid packet type %v", header.PType)

	case wspacket.Response:

	case wspacket.Notification:

	}
	return err
}

func handleSentPacket(header wspacket.Header) error {
	defer fmt.Printf("end %v\n", logdur.New("handleSentPacket"))
	return nil
}
