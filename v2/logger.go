package logging

import (
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"
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
	assert(l.h != nil, "logger not initialized")
	l.h.ch <- message{
		name: l.name,
		mesg: batch{
			text: l.fmt(mesg, args...),
			recv: time.Now(),
		},
	}
}

func (l Logger) Debug(mesg string, args ...interface{}) {
	err := exception{err: fmt.Errorf(mesg, args...)}
	switch l.h.mode {
	case LevelDebug:
		l.Print(err.Err().Error())
	case LevelTrace:
		err.Trace()
		l.Print(err.Error())
	}
}

func (l Logger) Trace(mesg string, args ...interface{}) {
	if l.h.mode == LevelTrace {
		l.Print(mesg, args...)
	}
}

func (l Logger) Catch(handler func(Logger, interface{})) {
	var err TracedError
	defer func() {
		if handler != nil {
			handler(l, err)
		} else if err != nil {
			l.Print("[CATCH] %v", err)
		}
	}()
	e := recover()
	if e == nil {
		return
	}
	switch te := e.(type) {
	case TracedError:
		err = te
	case error:
		err = &exception{err: te}
		err.Trace()
	default:
		err = &exception{err: fmt.Errorf("%v", e)}
		err.Trace()
	}
}

func (l Logger) Dump(data []byte, mesg string, args ...interface{}) {
	if l.h.mode < LevelDebug {
		return
	}
	l.Print(mesg, args...)
	if l.h.mode > LevelDebug {
		l.Print(strings.TrimSpace(hex.Dump(data)))
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

func (l Logger) Level() LogLevel {
	return l.h.mode
}
