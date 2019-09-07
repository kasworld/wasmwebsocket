// Copyright 2015,2016,2017,2018 SeukWon Kang (kasworld@gmail.com)

package wspacket

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"
)

const (
	HeaderLen      = 4 + 4 + 2 + 1 + 1
	MaxBodyLen     = 0xfffff
	MaxPacketLen   = HeaderLen + MaxBodyLen
	CompressLimit  = 0xffff
	EnableCompress = true
)

func (pk Packet) String() string {
	return fmt.Sprintf("Packet[%v %+v]", pk.Header, pk.Body)
}

type Packet struct {
	Header Header
	Body   interface{}
}

type PacketID uint32

const (
	PT_Request byte = iota
	PT_Response
	PT_Notification
)

type HeaderFlag uint8

const (
	HF_Compress HeaderFlag = iota
)

type Header struct {
	BodyLen uint32
	PkID    PacketID
	Cmd     uint16
	PType   byte
	Flags   byte
}

func MakeHeaderFromBytes(bytes []byte) Header {
	var header Header
	header.BodyLen = binary.LittleEndian.Uint32(bytes[0:4])
	header.PkID = PacketID(binary.LittleEndian.Uint32(bytes[4:8]))
	header.Cmd = binary.LittleEndian.Uint16(bytes[8:10])
	header.PType = bytes[10]
	header.Flags = bytes[11]
	return header
}

func GetBodyLenFromHeaderBytes(bytes []byte) uint32 {
	return binary.LittleEndian.Uint32(bytes[0:4])
}

func (v Header) ToBytes() []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, v)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}
	return buf.Bytes()
}

func (v *Header) GetByteSlice() []byte {
	return (*[HeaderLen]byte)(unsafe.Pointer(v))[:]
}

func (v *Header) SetFlag(pos HeaderFlag) {
	v.Flags |= (1 << pos)
}
func (v *Header) ClearFlag(pos HeaderFlag) {
	v.Flags &^= (1 << pos)
}
func (v *Header) NegFlag(pos HeaderFlag) {
	v.Flags ^= (1 << pos)
}
func (v *Header) GetFlag(pos HeaderFlag) bool {
	val := v.Flags & (1 << pos)
	return val != 0
}

func (h Header) String() string {
	switch h.PType {
	case PT_Request:
		return fmt.Sprintf(
			"Header[Req:%v BodyLen:%d PkID:%d  Flags:0b%0b]",
			h.Cmd,
			h.BodyLen,
			h.PkID,
			h.Flags,
		)
	case PT_Response:
		return fmt.Sprintf(
			"Header[Rsp:%v BodyLen:%d PkID:%d Flags:0b%0b]",
			h.Cmd,
			h.BodyLen,
			h.PkID,
			h.Flags,
		)
	case PT_Notification:
		return fmt.Sprintf(
			"Header[Noti:%v BodyLen:%d PkID:%d Flags:0b%0b]",
			h.Cmd,
			h.BodyLen,
			h.PkID,
			h.Flags,
		)
	default:
		return fmt.Sprintf(
			"Header[%v:%v BodyLen:%d PkID:%d  Flags:0b%0b]",
			h.PType,
			h.Cmd,
			h.BodyLen,
			h.PkID,
			h.Flags,
		)
	}
}

///////////////

func NewSendPacketBuffer() []byte {
	return make([]byte, MaxPacketLen)
}

type RecvPacketBuffer struct {
	RecvBuffer []byte
	RecvLen    int
}

func NewRecvPacketBuffer() *RecvPacketBuffer {
	pb := &RecvPacketBuffer{
		RecvBuffer: make([]byte, MaxPacketLen),
		RecvLen:    0,
	}
	return pb
}

func NewRecvPacketBufferByData(rdata []byte) *RecvPacketBuffer {
	pb := &RecvPacketBuffer{
		RecvBuffer: rdata,
		RecvLen:    len(rdata),
	}
	return pb
}

func (pb *RecvPacketBuffer) GetHeader() Header {
	if !pb.IsHeaderComplete() {
		return Header{}
	}
	header := MakeHeaderFromBytes(pb.RecvBuffer)
	return header
}

func (pb *RecvPacketBuffer) GetDecompressedBody() ([]byte, error) {
	if !pb.IsPacketComplete() {
		return nil, fmt.Errorf("packet not complete")
	}
	header := pb.GetHeader()
	body := pb.RecvBuffer[HeaderLen : HeaderLen+int(header.BodyLen)]
	if header.GetFlag(HF_Compress) {
		return DecompressData(body)
	}
	return body, nil
}

func (pb *RecvPacketBuffer) GetHeaderBody() (Header, []byte, error) {

	if !pb.IsPacketComplete() {
		return Header{}, nil, fmt.Errorf("packet not complete")
	}
	header := pb.GetHeader()
	body, err := pb.GetDecompressedBody()
	return header, body, err
}

func (pb *RecvPacketBuffer) IsHeaderComplete() bool {
	return pb.RecvLen >= HeaderLen
}

func (pb *RecvPacketBuffer) IsPacketComplete() bool {
	if !pb.IsHeaderComplete() {
		return false
	}
	bodylen := GetBodyLenFromHeaderBytes(pb.RecvBuffer)
	return pb.RecvLen == HeaderLen+int(bodylen)
}

func (pb *RecvPacketBuffer) NeedRecvLen() int {
	if !pb.IsHeaderComplete() {
		return HeaderLen
	}
	bodylen := GetBodyLenFromHeaderBytes(pb.RecvBuffer)
	return HeaderLen + int(bodylen)
}

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

var CompressData = compressZlib
var DecompressData = decompressZlib

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
