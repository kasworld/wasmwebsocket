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
	"fmt"
	"syscall/js"
	"time"

	"github.com/kasworld/wasmwebsocket/logdur"
	"github.com/kasworld/wasmwebsocket/wspacket"
)

var done chan struct{}

func main() {
	js.Global().Call("requestAnimationFrame", js.FuncOf(jsFrame))
	displayFrame()
	<-done
}

var lasttime time.Time

func jsFrame(js.Value, []js.Value) interface{} {
	tc := NewWebsocketConnection("ws://localhost:8080")
	err := tc.Connect()
	fmt.Printf("%v", err)

	displayFrame()
	js.Global().Call("requestAnimationFrame", js.FuncOf(jsFrame))
	return nil
}

func displayFrame() {
	fmt.Println("displayFrame")
	thistime := time.Now()
	if lasttime.Second() == thistime.Second() {
		return
	}
	lasttime = thistime
}

//////////////

type WebsocketConnection struct {
	remoteAddr   string
	conn         js.Value
	sendRecvStop func()
	sendCh       chan wspacket.Packet
}

func (tc *WebsocketConnection) String() string {
	return fmt.Sprintf("WebsocketConnection[%v SendCh:%v]",
		tc.remoteAddr, len(tc.sendCh))
}

func NewWebsocketConnection(connAddr string) *WebsocketConnection {
	tc := &WebsocketConnection{
		remoteAddr: connAddr,
		sendCh:     make(chan wspacket.Packet, 2),
	}
	tc.sendRecvStop = func() {
		fmt.Printf("Too early sendRecvStop call %v", tc)
	}
	return tc
}

func (tc *WebsocketConnection) Connect() error {
	defer fmt.Printf("end %v\n", logdur.New("ConnectTo"))

	tc.conn = js.Global().Get("WebSocket").New(tc.remoteAddr)
	if tc.conn == js.Null() {
		err := fmt.Errorf("fail to connect %v", tc.remoteAddr)
		fmt.Printf("%v", err)
		return err
	}
	tc.conn.Call("addEventListener", "message", js.FuncOf(tc.handleWebsocketMessage))
	connCtx, ctxCancel := context.WithCancel(context.Background())
	tc.sendRecvStop = ctxCancel
	go func() {
		err := WasmSendLoop(connCtx, tc.sendRecvStop, &tc.conn, tc.sendCh, tc.handleSentPacket)
		if err != nil {
			fmt.Printf("end SendLoop %v", err)
		}
	}()
	return nil
}

func WasmSendLoop(sendRecvCtx context.Context, SendRecvStop func(),
	conn *js.Value, SendCh chan wspacket.Packet,
	handleSentPacket func(header wspacket.Header) error,
) error {

	defer fmt.Printf("end %v\n", logdur.New("WasmSendLoop"))

	defer SendRecvStop()
	var err error
loop:
	for {
		select {
		case <-sendRecvCtx.Done():
			break loop
		case pk := <-SendCh:
			sendBuffer := make([]byte, wspacket.MaxPacketLen)
			sendN, err := wspacket.Packet2Bytes(&pk, sendBuffer, marshalBodyFn)
			if err != nil {
				break loop
			}
			if err = wasmSendPacket(conn, sendBuffer[:sendN]); err != nil {
				break loop
			}
			if err = handleSentPacket(pk.Header); err != nil {
				break loop
			}
		}
	}
	return err
}

func marshalBodyFn(body interface{}) ([]byte, error) {
	return json.Marshal(body)
}

func wasmSendPacket(conn *js.Value, sendBuffer []byte) error {
	defer fmt.Printf("end %v\n", logdur.New("wasmSendPacket"))

	sendData := js.Global().Get("Uint8Array").New(len(sendBuffer))
	js.CopyBytesToJS(sendData, sendBuffer)
	fmt.Println("conn", *conn, "sendData", sendData)
	rtn := conn.Call("send", sendData)
	if rtn != js.Null() {
		return fmt.Errorf("fail to send %v", rtn)
	}
	return nil
}

func (tc *WebsocketConnection) handleWebsocketMessage(this js.Value, args []js.Value) interface{} {
	defer fmt.Printf("end %v\n", logdur.New("handleWebsocketMessage"))

	fmt.Println(this, args)
	evt := args[0]
	fmt.Println(evt)
	rdata := make([]byte, wspacket.MaxPacketLen)
	rlen := js.CopyBytesToGo(rdata, evt.Get("data"))
	fmt.Println(rdata[:rlen])

	rPk := wspacket.NewRecvPacketBufferByData(rdata[:rlen])

	header, body, lerr := rPk.GetHeaderBody()
	if lerr != nil {
		fmt.Println(lerr)
		return nil
	} else {
		if err := tc.handleRecvPacket(header, body); err != nil {
			fmt.Println(err)
			return nil
		}
	}
	return nil
}

func (tc *WebsocketConnection) handleRecvPacket(header wspacket.Header, body []byte) error {
	defer fmt.Printf("end %v\n", logdur.New("handleRecvPacket"))

	var err error
	switch header.PType {
	default:
		err = fmt.Errorf("invalid packet type %v", header.PType)

	case wspacket.PT_Response:

	case wspacket.PT_Notification:

	}

	if err != nil {
		tc.sendRecvStop()
		return err
	}

	return nil
}

func (tc *WebsocketConnection) handleSentPacket(header wspacket.Header) error {
	defer fmt.Printf("end %v\n", logdur.New("handleSentPacket"))
	return nil
}

func (tc *WebsocketConnection) enqueueSendPacket(pk wspacket.Packet) error {
	defer fmt.Printf("end %v\n", logdur.New("enqueueSendPacket"))
	trycount := 10
	for trycount > 0 {
		select {
		case tc.sendCh <- pk:
			return nil
		default:
			trycount--
		}
		fmt.Printf("Send delayed, %s send channel busy %v, retry %v",
			tc, len(tc.sendCh), 10-trycount)
		time.Sleep(1 * time.Millisecond)
	}

	return fmt.Errorf("Send channel full %v", tc)
}
