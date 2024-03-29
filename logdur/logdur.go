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

package logdur

import (
	"fmt"
	"time"
)

type LogDur struct {
	name      string
	startTime time.Time
}

func (rd LogDur) String() string {
	return fmt.Sprintf("%v[%v]",
		rd.name,
		rd.GetElapsTime(),
	)
}

func New(name string) *LogDur {
	if name == "" {
		name = "LogDur"
	}
	fmt.Printf("begin %v\n", name)
	return &LogDur{
		name:      name,
		startTime: time.Now(),
	}
}

func (rd *LogDur) GetElapsTime() time.Duration {
	return time.Now().Sub(rd.startTime)
}
