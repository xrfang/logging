package logging

import (
	"fmt"
	"strings"
	"time"

	"github.com/xrfang/hxdump"
)

type (
	logger struct {
		name string
		mode int
		ch   chan message
	}
)

func (l logger) fmt(format string, args ...interface{}) []string {
	ts := time.Now().Format("2006-01-02 15:04:05 ")
	pad := strings.Repeat(" ", len(ts))
	var msg []string
	for i, m := range strings.Split(fmt.Sprintf(format, args...), "\n") {
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

func (l logger) Print(mesg string, args ...interface{}) {
	l.ch <- message{
		name: l.name,
		mesg: batch{
			text: l.fmt(mesg, args...),
			recv: time.Now(),
		},
	}
}

func (l logger) Debug(mesg string, args ...interface{}) {
	if l.mode < 1 {
		return
	}
	if l.mode > 1 {
		l.Print(trace(mesg, args...).Error())
	} else {
		l.Print(mesg, args...)
	}
}

func (l logger) Catch() {
	if e := recover(); e != nil {
		l.Print(trace("%v", e).Error())
	}
}

func (l logger) Dump(data []byte, mesg string, args ...interface{}) {
	if l.mode < 1 {
		return
	}
	l.Print(mesg, args...)
	if l.mode > 1 {
		l.Print(hxdump.DumpWithStyle(data, hxdump.Style{Narrow: true}))
	}
}

func (l logger) Flush() {
	l.ch <- message{name: l.name}
}
