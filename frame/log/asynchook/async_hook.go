package asynchook

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

var (
	// ErrCloesd ...
	ErrCloesd = errors.New("ErrCloesd")
)

// FilterFunc 返回 true 表示通过，false 表示被过滤。
type FilterFunc func(entry *logrus.Entry) bool

// WriteFunc 写入函数实现.
type WriteFunc func(entry *logrus.Entry) error

// Hook for logrus，异步写入 sql db，退出程序时调用 Close() 确保日志完全写入完毕。
type Hook struct {
	Filter FilterFunc

	levels []logrus.Level
	write  WriteFunc

	toWrite chan *logrus.Entry
	closed  int32
	wg      sync.WaitGroup
}

// New Hook, bufSize 表示异步缓冲任务队列的大小.
func New(bufSize int, levels []logrus.Level, write WriteFunc) *Hook {
	h := &Hook{
		levels:  levels,
		write:   write,
		toWrite: make(chan *logrus.Entry, bufSize),
	}

	go func() {
		for entry := range h.toWrite {
			err := h.write(entry)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to async log: %v\n", err)
			}
			h.wg.Done()
		}
	}()

	return h
}

// NewWithHook 将现有的 logrus.Hook 转换成异步形式.
func NewWithHook(bufSize int, hook logrus.Hook) *Hook {
	return New(bufSize, hook.Levels(), hook.Fire)
}

// Levels logrus.Hook interface
func (h *Hook) Levels() []logrus.Level {
	return h.levels
}

// Fire logrus.Hook interface
func (h *Hook) Fire(entry *logrus.Entry) error {
	if atomic.LoadInt32(&h.closed) != 0 {
		return ErrCloesd
	}
	if h.Filter != nil && !h.Filter(entry) {
		return nil
	}
	newEntry := logrus.NewEntry(entry.Logger)
	newEntry.Time = entry.Time
	newEntry.Level = entry.Level
	newEntry.Message = entry.Message
	for k, v := range entry.Data {
		newEntry.Data[k] = v
	}
	h.wg.Add(1)
	h.toWrite <- newEntry
	return nil
}

// Close 关闭异步记录循环, 该调用会等待所有操作完成.
func (h *Hook) Close() {
	atomic.StoreInt32(&h.closed, 1)
	close(h.toWrite)
	h.wg.Wait()
}
