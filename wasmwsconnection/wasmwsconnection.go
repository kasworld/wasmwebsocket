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
	"time"

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
		sendCh:             make(chan wspacket.Packet, 2),
		marshalBodyFn:      marshalBodyFn,
		handleRecvPacketFn: handleRecvPacketFn,
		handleSentPacketFn: handleSentPacketFn,
	}
	wsc.sendRecvStop = func() {
		jslog.Log("Too early sendRecvStop call %v", wsc)
	}
	return wsc
}

func (wsc *Connection) Connect() error {
	defer fmt.Printf("end %v\n", logdur.New("ConnectTo"))

	wsc.conn = js.Global().Get("WebSocket").New(wsc.remoteAddr)
	if wsc.conn == js.Null() {
		err := fmt.Errorf("fail to connect %v", wsc.remoteAddr)
		jslog.Log("%v", err)
		return err
	}
	wsc.conn.Call("addEventListener", "open", js.FuncOf(wsc.connected))
	return nil
}

func (wsc *Connection) connected(js.Value, []js.Value) interface{} {
	// wsc.conn.Set("binaryType", "arraybuffer")
	wsc.conn.Call("addEventListener", "message", js.FuncOf(wsc.handleWebsocketMessage))
	connCtx, ctxCancel := context.WithCancel(context.Background())
	wsc.sendRecvStop = ctxCancel
	go func() {
		err := wsc.sendLoop(connCtx)
		if err != nil {
			jslog.Log("end SendLoop %v", err)
		}
	}()
	return nil
}

func (wsc *Connection) sendLoop(sendRecvCtx context.Context) error {

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
			sendN, err := wspacket.Packet2Bytes(&pk, sendBuffer, wsc.marshalBodyFn)
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
	return err
}

func (wsc *Connection) sendPacket(sendBuffer []byte) error {
	defer fmt.Printf("end %v\n", logdur.New("sendPacket"))

	sendData := js.Global().Get("Uint8Array").New(len(sendBuffer))
	js.CopyBytesToJS(sendData, sendBuffer)
	fmt.Println("conn", wsc.conn, "sendData", sendData)
	rtn := wsc.conn.Call("send", sendData)
	if rtn != js.Null() {
		return fmt.Errorf("fail to send %v", rtn)
	}
	return nil
}

func (wsc *Connection) handleWebsocketMessage(this js.Value, args []js.Value) interface{} {
	defer fmt.Printf("end %v\n", logdur.New("handleWebsocketMessage"))

	fmt.Println(this, args)

	jslog.Log("arg0 [%v]\n", args[0])
	jslog.Log("arg0 data [%v]\n", args[0].Get("data"))
	jslog.Log("arg0 data len [%v]\n", args[0].Get("data").Get("size"))
	jslog.Log("arg0 data type [%v]\n", args[0].Get("data").Get("type"))

	// datalen := args[0].Get("data").Get("size").Int()
	data := args[0].Get("data") // blob
	jslog.Log("hello", data)
	// data := js.Global().Get("Uint8Array").New(args[0].Get("data"))
	// for i := 0; i < datalen; i++ {
	// 	fmt.Print(data.Index(i))
	// }
	jslog.Log("data [%v]\n", data)

	rdata := make([]byte, wspacket.MaxPacketLen)
	rlen := js.CopyBytesToGo(rdata, data)
	fmt.Println(rdata[:rlen])
	rPk := wspacket.NewRecvPacketBufferByData(rdata[:rlen])

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
}

func (wsc *Connection) EnqueueSendPacket(pk wspacket.Packet) error {
	defer fmt.Printf("end %v\n", logdur.New("EnqueueSendPacket"))
	trycount := 10
	for trycount > 0 {
		select {
		case wsc.sendCh <- pk:
			return nil
		default:
			trycount--
		}
		jslog.Log("Send delayed, %s send channel busy %v, retry %v",
			wsc, len(wsc.sendCh), 10-trycount)
		time.Sleep(1 * time.Millisecond)
	}

	return fmt.Errorf("Send channel full %v", wsc)
}
