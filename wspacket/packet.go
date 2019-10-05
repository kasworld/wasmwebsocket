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

package wspacket

import (
	"encoding/binary"
	"fmt"
	"io"
)

// packet flow type
const (
	// make not initalized packet error
	packetTypeError byte = iota

	// Request for request packet (response packet expected)
	Request

	// Response is reply of request packet
	Response

	// Notification is just send and forget packet
	Notification
)

const (
	// MaxBodyLen set to max body len, affect send/recv buffer size
	MaxBodyLen = 0xfffff

	// HeaderLen fixed size of header
	HeaderLen = 4 + 4 + 2 + 1 + 1 + 4

	// MaxPacketLen max total packet size byte of raw packet
	MaxPacketLen = HeaderLen + MaxBodyLen
)

func (pk Packet) String() string {
	return fmt.Sprintf("Packet[%v %+v]", pk.Header, pk.Body)
}

// Packet is header + body as object (not byte list)
type Packet struct {
	Header Header
	Body   interface{}
}

// Header is fixed size header of packet
type Header struct {
	bodyLen  uint32 // read only
	ID       uint32 // sender set, unique id per packet (wrap around reuse)
	Cmd      uint16 // sender set, application demux received packet
	FlowType byte   // sender set, flow control, Request, Response, Notification
	BodyType byte   // at marshal(Packet2Bytes) set, body compress, marshal type
	Fill     uint32 // sender set, any data
}

// MakeHeaderFromBytes unmarshal header from bytelist
func MakeHeaderFromBytes(buf []byte) Header {
	var h Header
	h.bodyLen = binary.LittleEndian.Uint32(buf[0:4])
	h.ID = binary.LittleEndian.Uint32(buf[4:8])
	h.Cmd = binary.LittleEndian.Uint16(buf[8:10])
	h.FlowType = buf[10]
	h.BodyType = buf[11]
	h.Fill = binary.LittleEndian.Uint32(buf[12:16])
	return h
}

// GetBodyLenFromHeaderBytes return packet body len from bytelist of header
func GetBodyLenFromHeaderBytes(buf []byte) uint32 {
	return binary.LittleEndian.Uint32(buf[0:4])
}

// ToByteList marshal header to bytelist
func (h Header) ToByteList() []byte {
	buf := make([]byte, HeaderLen)
	binary.LittleEndian.PutUint32(buf[0:4], h.bodyLen)
	binary.LittleEndian.PutUint32(buf[4:8], h.ID)
	binary.LittleEndian.PutUint16(buf[8:10], h.Cmd)
	buf[10] = h.FlowType
	buf[11] = h.BodyType
	binary.LittleEndian.PutUint32(buf[12:16], h.Fill)
	return buf
}

func (h Header) toBytesAt(buf []byte) {
	binary.LittleEndian.PutUint32(buf[0:4], h.bodyLen)
	binary.LittleEndian.PutUint32(buf[4:8], h.ID)
	binary.LittleEndian.PutUint16(buf[8:10], h.Cmd)
	buf[10] = h.FlowType
	buf[11] = h.BodyType
	binary.LittleEndian.PutUint32(buf[12:16], h.Fill)
}

// BodyLen return bodylen field
func (h *Header) BodyLen() uint32 {
	return h.bodyLen
}

func (h Header) String() string {
	return fmt.Sprintf(
		"Header[%d:%d ID:%d bodyLen:%d Compress:%v Fill:%d]",
		h.FlowType, h.Cmd, h.ID, h.bodyLen, h.BodyType, h.Fill)
}

///////////////

// func NewSendPacketBuffer() []byte {
// 	return make([]byte, MaxPacketLen)
// }

// func NewRecvPacketBuffer() *RecvPacketBuffer {
// 	pb := &RecvPacketBuffer{
// 		RecvBuffer: make([]byte, MaxPacketLen),
// 		RecvLen:    0,
// 	}
// 	return pb
// }

// RecvPacketBuffer used for packet receive
type RecvPacketBuffer struct {
	RecvBuffer []byte
	RecvLen    int
}

// NewRecvPacketBufferByData make RecvPacketBuffer by exist data
func NewRecvPacketBufferByData(rdata []byte) *RecvPacketBuffer {
	pb := &RecvPacketBuffer{
		RecvBuffer: rdata,
		RecvLen:    len(rdata),
	}
	return pb
}

// GetHeader make header and return
// if data is insufficent, return empty header
func (pb *RecvPacketBuffer) GetHeader() Header {
	if !pb.IsHeaderComplete() {
		return Header{}
	}
	header := MakeHeaderFromBytes(pb.RecvBuffer)
	return header
}

// GetBodyBytes return body ready to unmarshal.
// if body is BodyType, decompress and return
func (pb *RecvPacketBuffer) GetBodyBytes() ([]byte, error) {
	if !pb.IsPacketComplete() {
		return nil, fmt.Errorf("packet not complete")
	}
	header := pb.GetHeader()
	body := pb.RecvBuffer[HeaderLen : HeaderLen+int(header.bodyLen)]
	return body, nil
}

// GetHeaderBody return header and Body as bytelist
// application need demux by header.FlowType, header.Cmd
// unmarshal body with header.BodyType
// and check header.ID(if response packet)
func (pb *RecvPacketBuffer) GetHeaderBody() (Header, []byte, error) {
	if !pb.IsPacketComplete() {
		return Header{}, nil, fmt.Errorf("packet not complete")
	}
	header := pb.GetHeader()
	body, err := pb.GetBodyBytes()
	return header, body, err
}

// IsHeaderComplete check recv data is sufficient for header
func (pb *RecvPacketBuffer) IsHeaderComplete() bool {
	return pb.RecvLen >= HeaderLen
}

// IsPacketComplete check recv data is sufficient for packet
func (pb *RecvPacketBuffer) IsPacketComplete() bool {
	if !pb.IsHeaderComplete() {
		return false
	}
	bodylen := GetBodyLenFromHeaderBytes(pb.RecvBuffer)
	return pb.RecvLen == HeaderLen+int(bodylen)
}

// NeedRecvLen return need data len for header or body
func (pb *RecvPacketBuffer) NeedRecvLen() int {
	if !pb.IsHeaderComplete() {
		return HeaderLen
	}
	bodylen := GetBodyLenFromHeaderBytes(pb.RecvBuffer)
	return HeaderLen + int(bodylen)
}

// Read use for partial recv like tcp read
func (pb *RecvPacketBuffer) Read(conn io.Reader) error {
	toRead := pb.NeedRecvLen()
	for pb.RecvLen < toRead {
		n, err := conn.Read(pb.RecvBuffer[pb.RecvLen:toRead])
		if err != nil {
			return err
		}
		pb.RecvLen += n
	}
	return nil
}

// Packet2Bytes make packet to bytelist
// marshalBodyFn append marshaled(+compress) body to buffer and return total buffer, BodyType, error
// set Packet.Header.bodyLen, Packet.Header.BodyType
// return bytelist, error
func Packet2Bytes(pk *Packet, marshalBodyFn func(interface{}, []byte) ([]byte, byte, error)) ([]byte, error) {
	newbuf, bodytype, err := marshalBodyFn(pk.Body, make([]byte, HeaderLen, MaxPacketLen))
	if err != nil {
		return nil, err
	}
	bodyLen := len(newbuf) - HeaderLen
	if bodyLen > MaxBodyLen {
		return nil,
			fmt.Errorf("fail to serialize large packet %v, %v", pk.Header, bodyLen)
	}
	pk.Header.BodyType = bodytype
	pk.Header.bodyLen = uint32(bodyLen)
	pk.Header.toBytesAt(newbuf)
	return newbuf, nil
}
