package logstash

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	logrustash "github.com/bshuster-repo/logrus-logstash-hook"
	"github.com/sirupsen/logrus"
)

// Hook logstash hook for logrus
type Hook struct {
	conn   net.Conn
	impl   logrus.Hook
	ch     chan int
	closed int32

	rw sync.RWMutex
}

// New 创建一个 logstash hook
func New(addr string, fields logrus.Fields) (*Hook, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	h := new(Hook)
	h.conn = conn
	//h.impl, _ = logrustash.NewHookWithFieldsAndConn(conn, "appName", fields)
	h.impl = logrustash.New(conn, logrustash.DefaultFormatter(fields))
	h.ch = make(chan int, 0)

	go func() {
		for range h.ch {
			var conn net.Conn
			for {
				var err error
				conn, err = net.Dial("tcp", addr)
				if err == nil {
					break
				}
				if atomic.LoadInt32(&h.closed) != 0 {
					return
				}
				time.Sleep(0)
			}
			//impl, _ := logrustash.NewHookWithFieldsAndConn(conn, "appName", fields) //logrustash.New(conn, logrustash.DefaultFormatter(fields))
			impl := logrustash.New(conn, logrustash.DefaultFormatter(fields))
			if atomic.LoadInt32(&h.closed) != 0 {
				return
			}

			h.rw.Lock()
			h.conn = conn
			h.impl = impl
			h.rw.Unlock()
		}
	}()

	return h, nil
}

// Levels logrus.Hook interface
func (h *Hook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	}
}

// Fire logrus.Hook interface
func (h *Hook) Fire(entry *logrus.Entry) error {
	h.rw.RLock()
	defer h.rw.RUnlock()
	err := h.impl.Fire(entry)
	if err != nil {
		h.conn.Close()
		select {
		case h.ch <- 0:
		default:
		}
	}
	return err
}

// Close 关闭连接
func (h *Hook) Close() error {
	if atomic.CompareAndSwapInt32(&h.closed, 0, 1) {
		h.rw.RLock()
		defer h.rw.RUnlock()
		close(h.ch)
		return h.conn.Close()
	}
	return nil
}
