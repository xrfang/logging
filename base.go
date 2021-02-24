package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	LevelBrief = 0
	LevelDebug = 1
	LevelTrace = 2
)

type (
	LogLevel = int
	batch    struct {
		text []string
		recv time.Time
	}
	message struct {
		name string //保存的文件名
		mesg batch  //如果mesg为nil，表示强制写盘
	}
	Options struct {
		Split int         //切分LOG文件的尺寸，默认为10M
		Keep  int         //历史LOG文件保留数量，默认为10个
		Mode  os.FileMode //LOG目录的读写权限，默认为0755
		Cache int         //LOG在内存中缓存时长，默认为1秒，最短为1秒
		Queue int         //LOG队列长度，默认为64
		fMode os.FileMode //LOG文件的读写权限，由目录权限计算得出
	}
	logHandler struct {
		mode  LogLevel
		path  string
		opts  *Options
		cache map[string][]batch
		ch    chan message
		quit  chan bool
	}
)

func NewLogger(path string, mode LogLevel, opts *Options) (*logHandler, error) {
	if opts == nil {
		opts = new(Options)
	}
	if opts.Keep <= 0 {
		opts.Keep = 10
	}
	if opts.Mode == 0 {
		opts.Mode = 0755
	}
	opts.fMode = opts.Mode & 0666
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
		quit:  make(chan bool),
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
			if forceFlush && flushQueue == "" {
				break
			}
		}
		lh.quit <- true
	}()
	return &lh, nil
}

func (lh *logHandler) rotate(name string) {
	fmt.Printf("TODO: rotate '%s'...\n", name)
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
	defer func() {
		assert(f.Close())
		st, err := os.Stat(f.Name())
		assert(err)
		if st.Size() >= int64(lh.opts.Split) {
			old := fn + st.ModTime().Format(".2006-01-02_15.04.05")
			assert(os.Rename(fn, old))
			go lh.rotate(name)
		}
	}()
	for _, b := range logs {
		for _, line := range b.text {
			_, err = fmt.Fprintln(f, line)
			assert(err)
		}
	}
}

func (lh *logHandler) Flush() {
	lh.ch <- message{}
	<-lh.quit
}

func (lh *logHandler) Open(name string) logger {
	return logger{name: name, mode: lh.mode, ch: lh.ch}
}

var defaultLogHandler *logHandler

func Init(path string, mode LogLevel, opts *Options) (err error) {
	defaultLogHandler, err = NewLogger(path, mode, opts)
	return err
}

func Open(name string) logger {
	return defaultLogHandler.Open(name)
}

func Flush() {
	defaultLogHandler.Flush()
}
