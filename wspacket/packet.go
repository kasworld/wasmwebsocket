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
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// MaxBodyLen set to max body len, affect send/recv buffer size
	MaxBodyLen = 0xfffff

	// bodyCompressLimit is limit of uncompressed body size, compressed body can exceed this
	bodyCompressLimit = 0xffff

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

// packet type
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

// Compress is header flag to mark body compressed, read only
const (
	// make not initalized packet error
	compressError byte = iota

	// body not compressed
	CompressNone

	// body compressed by zlib
	CompressZlib
)

// Header is fixed size header of packet
type Header struct {
	bodyLen    uint32 // read only
	PkID       uint32 // sender set, for Request and Response
	Cmd        uint16 // sender set, application demux received packet
	PType      byte   // sender set, Request, Response, Notification
	compressed byte   // read only, body compressed state
	Fill       uint32 // sender set, any data
}

// MakeHeaderFromBytes unmarshal header from bytelist
func MakeHeaderFromBytes(buf []byte) Header {
	var h Header
	h.bodyLen = binary.LittleEndian.Uint32(buf[0:4])
	h.PkID = binary.LittleEndian.Uint32(buf[4:8])
	h.Cmd = binary.LittleEndian.Uint16(buf[8:10])
	h.PType = buf[10]
	h.compressed = buf[11]
	h.Fill = binary.LittleEndian.Uint32(buf[12:16])
	return h
}

// GetBodyLenFromHeaderBytes return packet body len from bytelist of header
func GetBodyLenFromHeaderBytes(buf []byte) uint32 {
	return binary.LittleEndian.Uint32(buf[0:4])
}

// toBytes marshal header to bytelist
func (h Header) toBytes() []byte {
	buf := make([]byte, HeaderLen)
	binary.LittleEndian.PutUint32(buf[0:4], h.bodyLen)
	binary.LittleEndian.PutUint32(buf[4:8], h.PkID)
	binary.LittleEndian.PutUint16(buf[8:10], h.Cmd)
	buf[10] = h.PType
	buf[11] = h.compressed
	binary.LittleEndian.PutUint32(buf[12:16], h.Fill)
	return buf
}

// BodyLen return bodylen field
func (h *Header) BodyLen() uint32 {
	return h.bodyLen
}

// Compress return compressed field
// func (h *Header) Compress() byte {
// 	return h.compressed
// }

// IsCompressed return is body compressed
func (h *Header) IsCompressed() bool {
	return h.compressed == CompressZlib
}

func (h Header) String() string {
	switch h.PType {
	case Request:
		return fmt.Sprintf(
			"Header[Req:%v bodyLen:%d PkID:%d Compress:%v Fill:0x%08x]",
			h.Cmd, h.bodyLen, h.PkID, h.IsCompressed(), h.Fill,
		)
	case Response:
		return fmt.Sprintf(
			"Header[Rsp:%v bodyLen:%d PkID:%d Compress:%v Fill:0x%08x]",
			h.Cmd, h.bodyLen, h.PkID, h.IsCompressed(), h.Fill,
		)
	case Notification:
		return fmt.Sprintf(
			"Header[Noti:%v bodyLen:%d PkID:%d Compress:%v Fill:0x%08x]",
			h.Cmd, h.bodyLen, h.PkID, h.IsCompressed(), h.Fill,
		)
	default:
		return fmt.Sprintf(
			"Header[%v:%v bodyLen:%d PkID:%d Compress:%v Fill:0x%08x]",
			h.PType, h.Cmd, h.bodyLen, h.PkID, h.IsCompressed(), h.Fill,
		)
	}
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

// GetDecompressedBody return body ready to unmarshal.
// if body is compressed, decompress and return
func (pb *RecvPacketBuffer) GetDecompressedBody() ([]byte, error) {
	if !pb.IsPacketComplete() {
		return nil, fmt.Errorf("packet not complete")
	}
	header := pb.GetHeader()
	body := pb.RecvBuffer[HeaderLen : HeaderLen+int(header.bodyLen)]
	if header.compressed == CompressZlib {
		return decompressZlib(body)
	}
	return body, nil
}

// GetHeaderBody return header and (decompressed) Body as bytelist
// application need demux by header.PType, header.Cmd,
// unmarshal body and check header.PkID(if response packet)
func (pb *RecvPacketBuffer) GetHeaderBody() (Header, []byte, error) {

	if !pb.IsPacketComplete() {
		return Header{}, nil, fmt.Errorf("packet not complete")
	}
	header := pb.GetHeader()
	body, err := pb.GetDecompressedBody()
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

/////////////////

func compressZlib(src []byte) ([]byte, error) {
	var b bytes.Buffer
	w, err := zlib.NewWriterLevel(&b, zlib.BestSpeed)
	if err != nil {
		return nil, err
	}
	w.Write(src)
	w.Close()
	return b.Bytes(), nil
}

func decompressZlib(src []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewBuffer(src))
	if err != nil {
		return nil, err
	}
	var dst bytes.Buffer
	io.Copy(&dst, r)
	r.Close()
	return dst.Bytes(), nil
}

// Packet2Bytes make packet to bytelist with compress(by bodyCompressLimit)
// marshalBodyFn is Packet.Body marshal function
// destBuffer must be allocated with enough size (MaxPacketLen)
// set Packet.Header.bodyLen to (compressed) body len
// return copied size, error
func Packet2Bytes(pk *Packet, destBuffer []byte,
	marshalBodyFn func(interface{}) ([]byte, error)) (int, error) {

	if len(destBuffer) < HeaderLen {
		return 0, fmt.Errorf("insufficient destBuffer %v < %v", len(destBuffer), HeaderLen)
	}
	bodyData, err := marshalBodyFn(pk.Body)
	if err != nil {
		return 0, err
	}
	if len(bodyData) > bodyCompressLimit {
		bodyData, err = compressZlib(bodyData)
		if err != nil {
			return 0, err
		}
		pk.Header.compressed = CompressZlib
	}
	bodyLen := len(bodyData)
	if bodyLen > MaxBodyLen {
		return 0,
			fmt.Errorf("fail to serialize large packet %v, %v", pk.Header, bodyLen)
	}
	pk.Header.bodyLen = uint32(bodyLen)
	copylen := copy(destBuffer, pk.Header.toBytes())
	copylen += copy(destBuffer[HeaderLen:], bodyData)
	if copylen != bodyLen+HeaderLen {
		return copylen, fmt.Errorf("insufficient destBuffer %v != %v", copylen, bodyLen+HeaderLen)
	}
	return copylen, nil
}
