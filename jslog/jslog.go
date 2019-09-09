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

package jslog

import "syscall/js"

func Error(v ...interface{}) {
	js.Global().Get("console").Call("error", v...)
}

func Warn(v ...interface{}) {
	js.Global().Get("console").Call("warn", v...)
}

func Debug(v ...interface{}) {
	js.Global().Get("console").Call("debug", v...)
}

func Info(v ...interface{}) {
	js.Global().Get("console").Call("info", v...)
}

func Log(v ...interface{}) {
	js.Global().Get("console").Call("log", v...)
}
