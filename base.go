package logging

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type (
	batch struct {
		text []string
		recv time.Time
	}
	message struct {
		name string //保存的文件名
		mesg batch  //如果mesg为nil，表示强制写盘
	}
	logOpts struct {
		Split int         //切分LOG文件的尺寸，默认为10M
		Keep  int         //历史LOG文件保留数量，默认为10个
		Mode  fs.FileMode //LOG目录的读写权限，默认为0755
		Cache int         //LOG在内存中缓存时长，默认为1秒，最短为1秒
		Queue int         //LOG队列长度，默认为64
		fMode fs.FileMode //LOG文件的读写权限，由目录权限计算得出
	}
	logHandler struct {
		mode  int //0=正常；1=debug；2=trace
		path  string
		opts  *logOpts
		cache map[string][]batch
		ch    chan message
	}
)

func NewLogger(path string, mode int, opts *logOpts) (*logHandler, error) {
	if opts == nil {
		opts = new(logOpts)
	}
	if opts.Keep <= 0 {
		opts.Keep = 10
	}
	if opts.Mode == 0 {
		opts.Mode = 0755
	}
	opts.fMode = opts.Mode & ^(0111)
	if opts.Split <= 0 {
		opts.Split = 10 * 1024 * 1024
	}
	if opts.Cache <= 1 {
		opts.Cache = 1
	}
	if opts.Queue <= 0 {
		opts.Queue = 64
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(path, opts.Mode)
	if err != nil {
		return nil, err
	}
	lh := logHandler{
		mode:  mode,
		path:  path,
		opts:  opts,
		cache: make(map[string][]batch),
		ch:    make(chan message, opts.Queue),
	}
	go func() {
		ttl := time.Duration(lh.opts.Cache) * time.Second
		timer := time.NewTimer(time.Second)
		for {
			forceFlush := false
			flushQueue := ""
			select {
			case msg := <-lh.ch:
				if len(msg.mesg.text) == 0 {
					forceFlush = true
					flushQueue = msg.name
				} else {
					lh.cache[msg.name] = append(lh.cache[msg.name], msg.mesg)
				}
			case <-timer.C:
			}
			for n, q := range lh.cache {
				if (forceFlush && (flushQueue == "" || flushQueue == n)) ||
					time.Since(q[0].recv) >= ttl {
					lh.flush(n)
				}
			}
		}
	}()
	return &lh, nil
}

func (lh *logHandler) rotate(name string) {

}

func (lh *logHandler) flush(name string) {
	logs := lh.cache[name]
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf(trace("logHandler.flush: %v", e).Error())
		}
		delete(lh.cache, name)
	}()
	fn := filepath.Join(lh.path, name)
	f, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, lh.opts.fMode)
	assert(err)
	defer func() { assert(f.Close()) }()
	for _, b := range logs {
		for _, line := b.text {
			_, err = fmt.Fprintln(f, line)
			assert(err)
		}
	}
	go lh.rotate(name)	
}

func (lh *logHandler) Flush() {
	lh.ch <- message{}
}

func (lh *logHandler) Open(name string) logger {
	return logger{name: name, mode: lh.mode, ch: lh.ch}
}
