package logging

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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
		mesg batch
		rply chan struct{} //一般消息的此属性为空，如果非空，表示强制写盘（此时忽略mesg）
	}
	Options struct {
		Split int         //切分LOG文件的尺寸，默认为10M
		Keep  int         //历史LOG文件保留数量，默认为10个
		Mode  os.FileMode //LOG目录的读写权限，默认为0755
		Cache int         //LOG在内存中缓存时长，默认为1秒，最短为1秒
		Queue int         //LOG队列长度，默认为64
		fMode os.FileMode //LOG文件的读写权限，由目录权限计算得出
	}
	LogHandler struct {
		mode  LogLevel
		path  string
		opts  *Options
		cache map[string][]batch
		ch    chan message
		wg    sync.WaitGroup
	}
)

func NewLogger(path string, mode LogLevel, opts *Options) (*LogHandler, error) {
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
	lh := LogHandler{
		mode:  mode,
		path:  path,
		opts:  opts,
		cache: make(map[string][]batch),
		ch:    make(chan message, opts.Queue),
	}
	go func() {
		ttl := time.Duration(lh.opts.Cache) * time.Second
		ticker := time.NewTicker(time.Second)
		var msg message
		for {
			select {
			case msg = <-lh.ch:
				if msg.rply == nil && len(msg.mesg.text) > 0 {
					lh.cache[msg.name] = append(lh.cache[msg.name], msg.mesg)
				}
			case <-ticker.C:
				msg = message{}
			}
			for n, q := range lh.cache {
				if (msg.rply != nil && (msg.name == n || msg.name == "")) || time.Since(q[0].recv) >= ttl {
					lh.flush(n)
				}
			}
			if msg.rply != nil { //是flush指令
				if msg.name == "" { //name属性为空表示需要结束整个logging组件
					break
				}
				close(msg.rply) //否则仅为flush一个指定的log文件
			}
		}
		lh.wg.Wait()
		close(msg.rply)
	}()
	return &lh, nil
}

func (lh *LogHandler) rotate(name string) {
	defer lh.wg.Done()
	oldLogs, _ := filepath.Glob(filepath.Join(lh.path, name+".*"))
	var backups []string
	for _, ol := range oldLogs {
		if strings.HasSuffix(ol, ".gz") {
			backups = append(backups, ol)
			continue
		}
		func(fn string) {
			defer func() {
				if e := recover(); e != nil {
					fmt.Fprintln(os.Stderr, trace("LogHandler.rotate: %v", e))
					return
				}
				os.Remove(fn)
				backups = append(backups, fn+".gz")
			}()
			f, err := os.Open(fn)
			assert(err)
			defer f.Close()
			g, err := os.Create(fn + ".gz")
			assert(err)
			defer func() { assert(g.Close()) }()
			zw := gzip.NewWriter(g)
			defer func() { assert(zw.Close()) }()
			_, err = io.Copy(zw, f)
			assert(err)
		}(ol)
	}
	sort.Slice(backups, func(i, j int) bool { return backups[i] < backups[j] })
	for len(backups) >= lh.opts.Keep {
		os.Remove(backups[0])
		backups = backups[1:]
	}
}

func (lh *LogHandler) flush(name string) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintln(os.Stderr, trace("LogHandler.flush: %v", e))
		}
		delete(lh.cache, name)
	}()
	fn := filepath.Join(lh.path, name)
	st, err := os.Stat(fn)
	if err == nil && st.Size() > int64(lh.opts.Split) {
		old := fn + st.ModTime().Format(".2006-01-02_15.04.05")
		assert(os.Rename(fn, old))
		lh.wg.Add(1)
		go lh.rotate(name)
	}
	f, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, lh.opts.fMode)
	assert(err)
	defer func() { assert(f.Close()) }()
	for _, b := range lh.cache[name] {
		for _, line := range b.text {
			_, err = fmt.Fprintln(f, line)
			assert(err)
		}
	}
}

func (lh *LogHandler) Path() string {
	return lh.path
}

func (lh *LogHandler) Close() {
	wait := make(chan struct{})
	lh.ch <- message{rply: wait}
	<-wait
}

func (lh *LogHandler) Open(name string) Logger {
	return Logger{name: name, h: lh}
}

var defaultLogHandler *LogHandler

func Init(path string, mode LogLevel, opts *Options) (err error) {
	defaultLogHandler, err = NewLogger(path, mode, opts)
	return err
}

func Open(name string) Logger {
	assert(defaultLogHandler != nil, "logging not initialized")
	return defaultLogHandler.Open(name)
}

func Path() string {
	assert(defaultLogHandler != nil, "logging not initialized")
	return defaultLogHandler.Path()
}

func Finish() {
	assert(defaultLogHandler != nil, "logging not initialized")
	defaultLogHandler.Close()
}
