package ws_handlereq

import (
	"context"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kasworld/wasmwebsocket/golog"
	"github.com/kasworld/wasmwebsocket/protocol/ws_error"
	"github.com/kasworld/wasmwebsocket/protocol/ws_json"
	"github.com/kasworld/wasmwebsocket/protocol/ws_loopwsgorilla"
	"github.com/kasworld/wasmwebsocket/protocol/ws_packet"
)

// service const
const (
	SendBufferSize = 10

	PacketReadTimeoutSec  = 6 * time.Second
	PacketWriteTimeoutSec = 3 * time.Second
)

func (c2sc *ServeClientConn) String() string {
	return fmt.Sprintf(
		"ServeClientConn[%v SendCh:%v]",
		c2sc.RemoteAddr,
		len(c2sc.sendCh),
	)
}

type ServeClientConn struct {
	RemoteAddr   string
	sendCh       chan ws_packet.Packet
	sendRecvStop func()
}

func NewServeClientConn(remoteAddr string) *ServeClientConn {
	c2sc := &ServeClientConn{
		RemoteAddr: remoteAddr,
		sendCh:     make(chan ws_packet.Packet, SendBufferSize),
	}

	c2sc.sendRecvStop = func() {
		golog.GlobalLogger.Fatal("Too early sendRecvStop call %v", c2sc)
	}
	return c2sc
}

func (c2sc *ServeClientConn) StartServeClientConn(mainctx context.Context, wsConn *websocket.Conn) {

	golog.GlobalLogger.Debug("Start ServeServeClientConn %s", c2sc)
	defer func() { golog.GlobalLogger.Debug("End ServeServeClientConn %s", c2sc) }()

	sendRecvCtx, sendRecvCancel := context.WithCancel(mainctx)
	c2sc.sendRecvStop = sendRecvCancel

	go func() {
		err := ws_loopwsgorilla.RecvLoop(sendRecvCtx, c2sc.sendRecvStop, wsConn,
			PacketReadTimeoutSec, c2sc.HandleRecvPacket)
		if err != nil {
			golog.GlobalLogger.Error("end RecvLoop %v", err)
		}
	}()
	go func() {
		err := ws_loopwsgorilla.SendLoop(sendRecvCtx, c2sc.sendRecvStop, wsConn,
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

func (c2sc *ServeClientConn) handleSentPacket(header ws_packet.Header) error {
	return nil
}

func (c2sc *ServeClientConn) HandleRecvPacket(header ws_packet.Header, rbody []byte) error {
	robj, err := ws_json.UnmarshalPacket(header, rbody)
	if err != nil {
		return err
	}
	if header.FlowType != ws_packet.Request {
		return fmt.Errorf("Unexpected header packet type: %v", header)
	}
	if int(header.Cmd) >= len(DemuxReq2APIFnMap) {
		return fmt.Errorf("Invalid header command %v", header)
	}
	response, errcode, apierr := DemuxReq2APIFnMap[header.Cmd](c2sc, header, robj)
	if errcode != ws_error.Disconnect && apierr == nil {
		rhd := header
		rhd.FlowType = ws_packet.Response
		rpk := ws_packet.Packet{
			Header: rhd,
			Body:   response,
		}
		c2sc.enqueueSendPacket(rpk)
	}
	return apierr
}

func (c2sc *ServeClientConn) enqueueSendPacket(pk ws_packet.Packet) error {
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
