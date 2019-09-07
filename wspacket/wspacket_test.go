package wspacket

import (
	"testing"
)

func TestHeader_ToBytes(t *testing.T) {
	hd := Header{
		Cmd:     3,
		BodyLen: 43545,
		PkID:    64,
		PType:   2,
		Flags:   23,
	}
	bs := hd.ToBytes()
	hd2 := MakeHeaderFromBytes(bs)
	t.Logf("%v %v %v", hd, bs, hd2)
}
