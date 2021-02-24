package logging

import (
	"fmt"
	"strings"
	"time"

	"github.com/xrfang/hxdump"
)

type (
	Logger struct {
		name string
		mode LogLevel
		ch   chan message
	}
)

func (l Logger) fmt(format string, args ...interface{}) []string {
	ts := time.Now().Format("2006-01-02 15:04:05 ")
	pad := strings.Repeat(" ", len(ts))
	var msg []string
	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}
	for i, m := range strings.Split(format, "\n") {
		m = strings.TrimRight(m, " \n\r\t")
		if m == "" {
			continue
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
	msg := message{
		name: l.name,
		mesg: batch{
			text: l.fmt(mesg, args...),
			recv: time.Now(),
		},
	}
	if len(msg.mesg.text) > 0 { //empty message indicates closure of logging facility
		l.ch <- msg
	}
}

func (l Logger) Debug(mesg string, args ...interface{}) {
	switch l.mode {
	case LevelDebug:
		l.Print(mesg, args...)
	case LevelTrace:
		l.Print(trace(mesg, args...).Error())
	}
}

func (l Logger) Catch(handler func(interface{})) {
	e := recover()
	if e != nil {
		l.Print(trace("%v", e).Error())
	}
	if handler != nil {
		handler(e)
	}
}

func (l Logger) Dump(data []byte, mesg string, args ...interface{}) {
	if l.mode < LevelDebug {
		return
	}
	l.Print(mesg, args...)
	if l.mode > LevelDebug {
		l.Print(hxdump.DumpWithStyle(data, hxdump.Style{Narrow: true}))
	}
}
