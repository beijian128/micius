package worker

import (
	"github.com/sirupsen/logrus"
	"github/beijian128/micius/frame/util"
	"sync"
	"sync/atomic"
)

type IWorker interface {
	// Post rpc 方式
	Post(f func())

	// Run 开启goroutine
	Run()
	// Fini 关闭
	Fini()
	// WorkerLen 负载
	WorkerLen() int32
}

type Worker struct {
	closed atomic.Bool
	finiWg sync.WaitGroup
	fs     chan func()
	len    atomic.Int32
}

func NewWorker(maxWorkerLen int) *Worker {
	w := &Worker{
		fs: make(chan func(), maxWorkerLen),
	}
	return w
}

// Post 传递f到goroutine上执行
func (w *Worker) Post(f func()) {
	w.fs <- f
}

func (w *Worker) Run() {
	w.finiWg.Add(1)

	go func() {
		defer util.Recover()
		defer func() {
			// 由于defer的调用比较耗内存，不能对外部的每个函数进行defer，所以采用了以下方式
			// 挂了重启，关了退出
			if w.closed.Load() {
				logrus.Error("IO catch err")
				w.Run()
			}
			w.finiWg.Done()
		}()
		for f := range w.fs {
			w.len.Add(1)
			f()
		}
	}()
}

func (w *Worker) WorkerLen() int32 {
	return w.len.Load()
}

func (w *Worker) Fini() {
	if w.closed.CompareAndSwap(false, true) {
		close(w.fs)
		w.finiWg.Wait()
	}
}
