package logging

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

type (
	TracedError interface {
		Err() error
		Error() string
		Stack() []string
		Describe(string, ...interface{})
		Trace()
	}
	exception struct {
		err   error
		msg   string
		trace []string
	}
)

func assert(e interface{}, ntfy ...interface{}) {
	var err *exception
	switch e.(type) {
	case nil:
	case bool:
		if !e.(bool) {
			mesg := "assertion failed"
			if len(ntfy) > 0 {
				mesg = ntfy[0].(string)
				if len(ntfy) > 1 {
					mesg = fmt.Sprintf(mesg, ntfy[1:]...)
				}
			}
			err = &exception{err: errors.New(mesg)}
		}
	case error:
		err = &exception{err: e.(error)}
	default:
		err = &exception{err: fmt.Errorf("assert: expect error or bool, got %T", e)}
	}
	if err != nil {
		err.Trace()
		panic(err)
	}
}

func (ex *exception) Trace() {
	if len(ex.trace) > 0 {
		return
	}
	n := 1
	for {
		n++
		pc, file, line, ok := runtime.Caller(n)
		if !ok {
			break
		}
		f := runtime.FuncForPC(pc)
		name := f.Name()
		if strings.HasPrefix(name, "runtime.") {
			continue
		}
		fn := strings.Split(file, "/")
		file = strings.Join(fn[len(fn)-2:], "/")
		ex.trace = append(ex.trace, fmt.Sprintf("(%s:%d) %s", file, line, name))
	}
}

func (ex *exception) Describe(msg string, args ...interface{}) {
	ex.msg = fmt.Sprintf(msg, args...)
}

func (ex exception) Err() error {
	return ex.err
}

func (ex exception) Error() string {
	msg := ex.msg
	if msg == "" {
		msg = ex.Err().Error()
	}
	stack := []string{msg}
	for _, t := range ex.trace {
		stack = append(stack, "\t"+t)
	}
	return strings.Join(stack, "\n")
}

func (ex exception) Stack() []string {
	return ex.trace
}

func trace(args ...interface{}) *exception {
	if len(args) == 0 {
		return nil
	}
	var ex exception
	switch args[0].(type) {
	case string:
		ex.err = fmt.Errorf(args[0].(string), args[1:]...)
	case error:
		if len(args) > 1 {
			ex.err = errors.New("trace: extra argument for error")
		} else {
			ex.err = args[0].(error)
		}
	default:
		ex.err = fmt.Errorf("trace: invalid type for arg[0] (%T)", args[0])
	}
	n := 1
	for {
		n++
		pc, file, line, ok := runtime.Caller(n)
		if !ok {
			break
		}
		f := runtime.FuncForPC(pc)
		name := f.Name()
		if strings.HasPrefix(name, "runtime.") {
			continue
		}
		fn := strings.Split(file, "/")
		file = strings.Join(fn[len(fn)-2:], "/")
		ex.trace = append(ex.trace, fmt.Sprintf("(%s:%d) %s", file, line, name))
	}
	return &ex
}
