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
		Trace()
	}
	tracedError struct {
		err   error
		trace []string
	}
)

func assert(e interface{}, ntfy ...interface{}) {
	var err *tracedError
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
			err = &tracedError{err: errors.New(mesg)}
		}
	case error:
		err = &tracedError{err: e.(error)}
	default:
		err = &tracedError{err: fmt.Errorf("assert: expect error or bool, got %T", e)}
	}
	if err != nil {
		err.Trace()
		panic(err)
	}
}

func (te *tracedError) Trace() {
	if len(te.trace) > 0 {
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
		te.trace = append(te.trace, fmt.Sprintf("(%s:%d) %s", file, line, name))
	}
}

func (te tracedError) Err() error {
	return te.err
}

func (te tracedError) Error() string {
	stack := []string{te.Err().Error()}
	for _, t := range te.trace {
		stack = append(stack, "\t"+t)
	}
	return strings.Join(stack, "\n")
}

func (te tracedError) Stack() []string {
	return te.trace
}

func trace(args ...interface{}) *tracedError {
	if len(args) == 0 {
		return nil
	}
	var te tracedError
	switch args[0].(type) {
	case string:
		te.err = fmt.Errorf(args[0].(string), args[1:]...)
	case error:
		if len(args) > 1 {
			te.err = errors.New("trace: extra argument for error")
		} else {
			te.err = args[0].(error)
		}
	default:
		te.err = fmt.Errorf("trace: invalid type for arg[0] (%T)", args[0])
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
		te.trace = append(te.trace, fmt.Sprintf("(%s:%d) %s", file, line, name))
	}
	return &te
}
