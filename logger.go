package logging

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/xrfang/hxdump"
)

type (
	Logger struct {
		name string
		h    *LogHandler
	}
)

func (l Logger) fmt(format string, args ...interface{}) []string {
	ts := time.Now().Format("2006-01-02 15:04:05 ")
	pad := strings.Repeat(" ", len(ts))
	var msg []string
	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}
	gap := false //used to remove excessive empty linese
	for i, m := range strings.Split(format, "\n") {
		m = strings.TrimRight(m, " \n\r\t")
		if m == "" {
			if gap {
				continue
			}
			gap = true
		} else {
			gap = false
		}
		if i == 0 {
			msg = append(msg, ts+m)
		} else {
			msg = append(msg, pad+m)
		}
	}
	return msg
}

func (l Logger) Print(mesg string, args ...interface{}) {
	l.h.ch <- message{
		name: l.name,
		mesg: batch{
			text: l.fmt(mesg, args...),
			recv: time.Now(),
		},
	}
}

func (l Logger) Debug(mesg string, args ...interface{}) {
	switch l.h.mode {
	case LevelDebug:
		l.Print(mesg, args...)
	case LevelTrace:
		l.Print(trace(mesg, args...).Error())
	}
}

func (l Logger) Catch(handler func(Logger, interface{})) {
	e := recover()
	if e != nil {
		l.Print(trace("%v", e).Error())
	}
	if handler != nil {
		handler(l, e)
	}
}

func (l Logger) Dump(data []byte, mesg string, args ...interface{}) {
	if l.h.mode < LevelDebug {
		return
	}
	l.Print(mesg, args...)
	if l.h.mode > LevelDebug {
		l.Print(hxdump.DumpWithStyle(data, hxdump.Style{Narrow: true}))
	}
}

func (l Logger) Flush() {
	wait := make(chan struct{})
	l.h.ch <- message{name: l.name, rply: wait}
	<-wait
}

func (l Logger) Path() string {
	return filepath.Join(l.h.Path(), l.name)
}
