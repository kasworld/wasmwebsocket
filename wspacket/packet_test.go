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
