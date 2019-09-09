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

package wasmwsconnection

import (
	"context"
	"fmt"
	"syscall/js"

	"github.com/kasworld/wasmwebsocket/jslog"
	"github.com/kasworld/wasmwebsocket/logdur"
	"github.com/kasworld/wasmwebsocket/wspacket"
)

type Connection struct {
	remoteAddr   string
	conn         js.Value
	sendRecvStop func()
	sendCh       chan wspacket.Packet

	marshalBodyFn      func(interface{}) ([]byte, error)
	handleRecvPacketFn func(header wspacket.Header, body []byte) error
	handleSentPacketFn func(header wspacket.Header) error
}

func (wsc *Connection) String() string {
	return fmt.Sprintf("Connection[%v SendCh:%v]",
		wsc.remoteAddr, len(wsc.sendCh))
}

func New(
	connAddr string,
	marshalBodyFn func(interface{}) ([]byte, error),
	handleRecvPacketFn func(header wspacket.Header, body []byte) error,
	handleSentPacketFn func(header wspacket.Header) error,
) *Connection {
	wsc := &Connection{
		remoteAddr:         connAddr,
		sendCh:             make(chan wspacket.Packet, 10),
		marshalBodyFn:      marshalBodyFn,
		handleRecvPacketFn: handleRecvPacketFn,
		handleSentPacketFn: handleSentPacketFn,
	}
	wsc.sendRecvStop = func() {
		fmt.Printf("Too early sendRecvStop call %v", wsc)
	}
	return wsc
}

func (wsc *Connection) Connect() error {
	defer fmt.Printf("end %v\n", logdur.New("ConnectTo"))

	wsc.conn = js.Global().Get("WebSocket").New(wsc.remoteAddr)
	if wsc.conn == js.Null() {
		err := fmt.Errorf("fail to connect %v", wsc.remoteAddr)
		fmt.Printf("%v", err)
		return err
	}
	wsc.conn.Call("addEventListener", "open", js.FuncOf(wsc.wsOpened))
	wsc.conn.Call("addEventListener", "close", js.FuncOf(wsc.wsClosed))
	wsc.conn.Call("addEventListener", "error", js.FuncOf(wsc.wsError))
	return nil
}

func (wsc *Connection) wsOpened(this js.Value, args []js.Value) interface{} {
	wsc.conn.Call("addEventListener", "message", js.FuncOf(wsc.handleWebsocketMessage))
	connCtx, ctxCancel := context.WithCancel(context.Background())
	wsc.sendRecvStop = ctxCancel
	go wsc.sendLoop(connCtx)
	return nil
}

func (wsc *Connection) wsClosed(this js.Value, args []js.Value) interface{} {
	wsc.sendRecvStop()
	jslog.Error("ws closed")
	return nil
}

func (wsc *Connection) wsError(this js.Value, args []js.Value) interface{} {
	wsc.sendRecvStop()
	jslog.Error(this, args)
	return nil
}

func (wsc *Connection) sendLoop(sendRecvCtx context.Context) {
	defer fmt.Printf("end %v\n", logdur.New("sendLoop"))
	defer wsc.sendRecvStop()
	var err error
loop:
	for {
		select {
		case <-sendRecvCtx.Done():
			break loop
		case pk := <-wsc.sendCh:
			sendBuffer := make([]byte, wspacket.MaxPacketLen)
			sendN := 0
			sendN, err = wspacket.Packet2Bytes(&pk, sendBuffer, wsc.marshalBodyFn)
			if err != nil {
				break loop
			}
			if err = wsc.sendPacket(sendBuffer[:sendN]); err != nil {
				break loop
			}
			if err = wsc.handleSentPacketFn(pk.Header); err != nil {
				break loop
			}
		}
	}
	fmt.Printf("end SendLoop %v\n", err)
	return
}

func (wsc *Connection) sendPacket(sendBuffer []byte) error {
	defer fmt.Printf("end %v\n", logdur.New("sendPacket"))

	sendData := js.Global().Get("Uint8Array").New(len(sendBuffer))
	js.CopyBytesToJS(sendData, sendBuffer)
	wsc.conn.Call("send", sendData)

	return nil
}

func (wsc *Connection) handleWebsocketMessage(this js.Value, args []js.Value) interface{} {
	defer fmt.Printf("end %v\n", logdur.New("handleWebsocketMessage"))

	data := args[0].Get("data") // blob
	aBuff := data.Call("arrayBuffer")
	aBuff.Call("then",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {

			rdata := ArrayBufferToSlice(args[0])
			rPk := wspacket.NewRecvPacketBufferByData(rdata)
			header, body, lerr := rPk.GetHeaderBody()
			if lerr != nil {
				fmt.Println(lerr)
				return nil
			} else {
				if err := wsc.handleRecvPacketFn(header, body); err != nil {
					fmt.Println(err)
					wsc.sendRecvStop()
					return nil
				}
			}

			return nil
		}))

	return nil
}

func Uint8ArrayToSlice(value js.Value) []byte {
	s := make([]byte, value.Get("byteLength").Int())
	js.CopyBytesToGo(s, value)
	return s
}

func ArrayBufferToSlice(value js.Value) []byte {
	return Uint8ArrayToSlice(js.Global().Get("Uint8Array").New(value))
}

func (wsc *Connection) EnqueueSendPacket(pk wspacket.Packet) error {
	defer fmt.Printf("end %v\n", logdur.New("EnqueueSendPacket"))
	select {
	case wsc.sendCh <- pk:
		return nil
	default:
		return fmt.Errorf("Send channel full %v", wsc)
	}
}
