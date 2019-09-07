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

package golog

import (
	"bytes"
	"fmt"
	"math/bits"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kasworld/log/logdestination_file"
	"github.com/kasworld/log/logdestination_stdio"
	"github.com/kasworld/log/logdestinationgroup"
	"github.com/kasworld/log/logdestinationi"
	"github.com/kasworld/log/logflagi"
	"github.com/kasworld/log/logflags"
)

type LL_Type uint64

func (ll LL_Type) ToShiftedNum() int {
	return bits.Len(uint(ll)) - 1
}

func (ll LL_Type) LevelsString() string {
	var buff bytes.Buffer

	buff.WriteString("LL_Type[")
	for i := LL_Type(1); i < LL_END; i <<= 1 {
		if ll.IsLevel(i) {
			fmt.Fprintf(&buff, "%s, ", i)
		}
	}
	buff.WriteString("]")
	return buff.String()
}

func (ll LL_Type) IsLevel(level LL_Type) bool {
	return ll&level != 0
}

func (i LL_Type) String() string {
	if str, ok := leveldata[i]; ok {
		return str
	}
	return "LL_Type(" + strconv.FormatInt(int64(i), 10) + ")"
}

///////////////////////////////////////////////////////////////////////////////////////////

var (
	OutputStdout = logdestination_stdio.NewStdOut()
	OutputStderr = logdestination_stdio.NewStdErr()
)

type logDestInfo struct {
	refCntByLogLv int // count referenced by each loglv
	dest          logdestinationi.LogDestinationI
}

type LogBase struct {
	mutex sync.RWMutex

	flag     logflagi.LogFlagI // properties
	prefix   string            // prefix to write at beginning of each line
	loglevel LL_Type

	ltype2destgrp     []*logdestinationgroup.LogDestinationGroup
	allDestInfoByName map[string]*logDestInfo
}

func New(prefix string, lf logflagi.LogFlagI, lv LL_Type) *LogBase {

	maxlen := LL_END.ToShiftedNum()
	dstgrp := make([]*logdestinationgroup.LogDestinationGroup, maxlen)
	for i := 0; i < maxlen; i++ {
		dstgrp[i] = logdestinationgroup.New()
	}

	return &LogBase{
		ltype2destgrp:     dstgrp,
		allDestInfoByName: make(map[string]*logDestInfo),
		flag:              lf,
		prefix:            prefix,
		loglevel:          lv,
	}
}

func NewWithDstDir(prefix string, logdir string, lf logflagi.LogFlagI,
	loglevel LL_Type, splitLogLevel LL_Type) (*LogBase, error) {
	logdir = strings.TrimSpace(logdir)
	if logdir == "" {
		return nil, fmt.Errorf("logdir empty %v", logdir)
	}
	if fileinfo, err := os.Stat(logdir); err != nil { // PathError
		if mkerr := os.Mkdir(logdir, os.ModePerm); mkerr != nil {
			return nil, mkerr
		}
	} else if fileinfo.IsDir() == false {
		return nil, fmt.Errorf("not a directory %v", logdir)
	}

	basename := filepath.Base(logdir)
	newlg := New(prefix, lf, loglevel)

	fnameForOther := fmt.Sprintf("%s.%s.%s", basename, "Other", "log")
	fpathForOther := filepath.Join(logdir, fnameForOther)
	newDestForOther, err := logdestination_file.New(fpathForOther)
	if err != nil {
		return nil, err
	}
	newlg.AddDestination(LL_All^splitLogLevel, newDestForOther)

	for ll := LL_Type(1); ll < LL_END; ll <<= 1 {
		if splitLogLevel&ll == ll {
			fnameForLL := fmt.Sprintf("%s.%s.%s", basename, ll.String(), "log")
			fpathForLL := filepath.Join(logdir, fnameForLL)
			newDestForLL, serr := logdestination_file.New(fpathForLL)
			if serr != nil {
				return nil, serr
			}
			newlg.AddDestination(ll, newDestForLL)
		}
	}
	newlg.AddDestination(LL_Fatal, OutputStdout)
	newlg.AddDestination(LL_Fatal, OutputStderr)
	return newlg, nil
}

func (lg *LogBase) AddDestination(
	ll LL_Type, o logdestinationi.LogDestinationI) {

	lg.mutex.Lock()
	defer lg.mutex.Unlock()

	for i := 0; i < len(lg.ltype2destgrp); i++ {
		s := LL_Type(1 << uint(i))
		if ll&s == 0 {
			continue
		}
		lg.addDestination1DestGrp(i, o)
	}
}

func (lg *LogBase) addDestination1DestGrp(
	i int, o logdestinationi.LogDestinationI) {

	added := lg.ltype2destgrp[i].AddDestination(o)
	if !added {
		if _, ok := lg.allDestInfoByName[o.Name()]; !ok {
			panic(fmt.Sprintf(
				"%v failed to AddDestination to destgroup index:%v, abnormal state",
				lg,
				i))
		}
		fmt.Printf("%v not added to destgroup index:%v\n", o, i)
		return
	}

	if dstinfo, ok := lg.allDestInfoByName[o.Name()]; ok {
		dstinfo.refCntByLogLv++
	} else {
		lg.allDestInfoByName[o.Name()] = &logDestInfo{
			refCntByLogLv: 1,
			dest:          o,
		}
	}
}

func (lg *LogBase) DelDestination(
	ll LL_Type, o logdestinationi.LogDestinationI) {

	lg.mutex.Lock()
	defer lg.mutex.Unlock()

	for i := 0; i < len(lg.ltype2destgrp); i++ {
		s := LL_Type(1 << uint(i))
		if ll&s == 0 {
			continue
		}
		lg.delDestinationFrom1DestGrp(i, o)
	}
}

func (lg *LogBase) delDestinationFrom1DestGrp(
	i int, o logdestinationi.LogDestinationI) {

	deleted := lg.ltype2destgrp[i].DelDestination(o)
	if !deleted {
		fmt.Printf("%v not deleted from destgroup index:%v\n", o, i)
		return
	}

	if dstinfo, ok := lg.allDestInfoByName[o.Name()]; ok {
		dstinfo.refCntByLogLv--
		if dstinfo.refCntByLogLv <= 0 {
			delete(lg.allDestInfoByName, o.Name())
		}
	} else {
		panic(fmt.Sprintf(
			"%v failed to DelDestination %v from destgroup index:%v, abnormal state",
			lg,
			o,
			i,
		))
	}
}

func (lg *LogBase) Reload() error {
	lg.mutex.RLock()
	defer lg.mutex.RUnlock()

	for _, v := range lg.allDestInfoByName {
		if err := v.dest.Reload(); err != nil {
			fmt.Println(err)
		}
	}
	return nil
}

func (lg *LogBase) LogPrintf(
	calldepth int, ll LL_Type,
	format string, v ...interface{}) ([]byte, error) {
	s := lg.Format2Bytes(calldepth+1, ll, format, v...)
	err := lg.Output(ll, s)
	return s, err
}

func (lg *LogBase) Format2Bytes(
	calldepth int, ll LL_Type,
	format string, v ...interface{}) []byte {

	if !lg.IsLevel(ll) {
		return nil
	}
	s := fmt.Sprintf(format, v...)

	var buf []byte
	llinfo := fmt.Sprintf("%s", ll)
	lg.flag.FormatHeader(&buf, calldepth+2, time.Now(), lg.prefix, llinfo)
	buf = append(buf, s...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf = append(buf, '\n')
	}
	return buf
}

func (lg *LogBase) Output(ll LL_Type, b []byte) error {
	i := ll.ToShiftedNum()
	return lg.ltype2destgrp[i].Write(b)
}

func (lg *LogBase) Panic(format string, v ...interface{}) error {
	s, err := lg.LogPrintf(2, LL_Fatal, format, v...)
	panic(string(s))
	return err
}

func (lg *LogBase) AddLevel(level LL_Type) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()
	lg.loglevel |= level
}

func (lg *LogBase) SetLevel(level LL_Type) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()
	lg.loglevel = level
}

func (lg *LogBase) DelLevel(level LL_Type) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()
	lg.loglevel &= ^level
}

func (lg *LogBase) IsLevel(level LL_Type) bool {
	return lg.loglevel&level != 0
}

func (lg *LogBase) FlagString() string {
	return lg.flag.FlagString()
}

func (lg *LogBase) LevelString() string {
	return lg.loglevel.LevelsString()
}

func (lg *LogBase) SetPrefix(p string) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()
	lg.prefix = p
}

// Prefix returns the output prefix for the logger.
func (lg *LogBase) GetPrefix() string {
	return lg.prefix
}

// Flags returns the output flags for the logger.
func (lg *LogBase) GetFlags() logflagi.LogFlagI {
	return lg.flag
}

// SetFlags sets the output flags for the logger.
func (lg *LogBase) SetFlags(flag logflagi.LogFlagI) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()
	lg.flag = flag
}

func (lg LogBase) String() string {
	return fmt.Sprintf("LogBase[%v %v]",
		lg.FlagString(), lg.LevelString(),
	)
}

//////////////////////////////////////////////////////////////////

var GlobalLogger = New("", logflags.DefaultValue(false), LL_All)

func init() {
	GlobalLogger.AddDestination(LL_All, OutputStderr)
}

func Reload() error {
	return GlobalLogger.Reload()
}

func GetLogger() *LogBase {
	return GlobalLogger
}
func SetLogger(l *LogBase) {
	GlobalLogger = l
}

func LevelString() string {
	return GlobalLogger.String()
}

func AddLevel(level LL_Type) {
	GlobalLogger.AddLevel(level)
}

func SetLevel(level LL_Type) {
	GlobalLogger.SetLevel(level)
}

func DelLevel(level LL_Type) {
	GlobalLogger.DelLevel(level)
}

func IsLevel(level LL_Type) bool {
	return GlobalLogger.IsLevel(level)
}

func SetPrefix(p string) {
	GlobalLogger.SetPrefix(p)
}

// Prefix returns the output prefix for the GlobalLogger.
func GetPrefix() string {
	return GlobalLogger.GetPrefix()
}

// Flags returns the output flags for the GlobalLogger.
func GetFlags() logflagi.LogFlagI {
	return GlobalLogger.GetFlags()
}

// SetFlags sets the output flags for the GlobalLogger.
func SetFlags(flag logflagi.LogFlagI) {
	GlobalLogger.SetFlags(flag)
}

func Panic(format string, v ...interface{}) error {
	s, err := GlobalLogger.LogPrintf(2, LL_Fatal, format, v...)
	panic(string(s))
	return err
}

func SetLog(logdir string, loglevel LL_Type, splitLogLevel LL_Type) error {
	if logdir != "" {
		newlg, err := NewWithDstDir(
			"",
			logdir,
			logflags.DefaultValue(false),
			loglevel,
			splitLogLevel,
		)
		if err != nil {
			return err
		}
		SetLogger(newlg)
	} else {
		SetLevel(loglevel)
	}
	return nil
}
