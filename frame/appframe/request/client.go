package request

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// TimeForever 表示请求忽略超时时间.
const TimeForever time.Duration = time.Duration(1<<63 - 1)

var (
	// ErrTimeOut 超时错误
	ErrTimeOut = errors.New("ErrTimeOut")
	// ErrCancel 取消请求.
	ErrCancel = errors.New("ErrCancel")
)

type handler func(resp any, err error)

// Client 发起请求并获取响应, 支持同步风格, 和一步回调风格.
type Client struct {
	worker   func(f func())
	waits    map[int64]handler
	seqid    int64
	mtx      sync.Mutex
	chanPool sync.Pool
	wg       sync.WaitGroup

	OnNotFind func(seqid int64, resp any, err error)
}

// NewClient 创建一个 Client
func NewClient(worker func(f func())) *Client {
	result := &Client{
		worker: worker,
		waits:  make(map[int64]handler),
	}
	result.chanPool.New = func() any {
		return make(chan reqResult, 1)
	}
	//race 竞争fix
	result.wg.Add(1)
	return result
}

func (c *Client) add(f handler) int64 {
	c.wg.Add(1)
	c.mtx.Lock()
	var seq int64
	for {
		c.seqid++
		seq = c.seqid
		if _, exist := c.waits[seq]; !exist && seq != 0 {
			break
		}
	}
	c.waits[seq] = f
	c.mtx.Unlock()

	//logrus.Debugf("add  seqid= %v   f=%v", seq, runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name())
	return seq
}

func (c *Client) remove(seq int64) (handler, bool) {
	c.mtx.Lock()
	f, ok := c.waits[seq]
	if ok {
		delete(c.waits, seq)
	}
	c.mtx.Unlock()
	//logrus.Debugf("remove  seqid= %v   f=%v", seq, runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name())
	return f, ok
}

func (c *Client) after(d time.Duration, f func()) (cancel func()) {
	if d == TimeForever {
		return func() {}
	}
	var stop int32
	f2 := func() {
		if atomic.LoadInt32(&stop) == 0 {
			f()
		}
	}
	t := time.AfterFunc(d, func() {
		if c.worker != nil {
			c.worker(f2)
		} else {
			f2()
		}
	})
	return func() {
		atomic.StoreInt32(&stop, 1)
		t.Stop()
	}
}

// OnErr 响应错误.
func (c *Client) OnErr(seqid int64, err error) {
	f, ok := c.remove(seqid)
	if ok {
		f(nil, err)
	} else {
		if !errors.Is(err, ErrCancel) {
			c.OnNotFind(seqid, nil, err)
		}
	}
}

// OnErrAll 响应所有请求错误.
func (c *Client) OnErrAll(err error) {
	c.mtx.Lock()
	waits := c.waits
	c.waits = make(map[int64]handler)
	c.mtx.Unlock()
	for _, f := range waits {
		f(nil, err)
	}
}

// OnResp 响应结果.
func (c *Client) OnResp(seqid int64, resp any) {
	f, ok := c.remove(seqid)
	if ok {
		f(resp, nil)
	} else {
		c.OnNotFind(seqid, resp, nil)
	}
}

// Req 请求并获取响应, 异步回调.
// 返回值为取消等待响应的操作接口.
func (c *Client) Req(req func(seqid int64) error, cbk func(resp any, err error), timeout time.Duration) (cancel func()) {
	var stopTimer func()
	seqid := c.add(func(resp any, err error) {
		if stopTimer != nil {
			stopTimer()
		}
		cbk(resp, err)
		c.wg.Done()
	})
	stopTimer = c.after(timeout, func() {
		c.OnErr(seqid, ErrTimeOut)
	})
	err := req(seqid)
	if err != nil {
		c.after(0, func() {
			c.OnErr(seqid, err)
		})
	}
	return func() {
		c.OnErr(seqid, ErrCancel)
	}
}

type reqResult struct {
	resp any
	err  error
}

// Call 请求并获得响应, 同步阻塞.
func (c *Client) Call(req func(seqid int64) error, timeout time.Duration) (any, error) {
	ch := c.chanPool.Get().(chan reqResult)
	seqid := c.add(func(resp any, err error) {
		ch <- reqResult{
			resp: resp,
			err:  err,
		}
	})
	cancelTimer := c.after(timeout, func() {
		c.OnErr(seqid, ErrTimeOut)
	})
	err := req(seqid)
	if err != nil {
		c.after(0, func() {
			c.OnErr(seqid, err)
		})
	}
	result := <-ch
	c.chanPool.Put(ch)
	cancelTimer()
	c.wg.Done()
	return result.resp, result.err
}

// WaitAllDone 等待所有响应处理结束.
func (c *Client) WaitAllDone() {
	c.wg.Done() // 对应 NewClient wg.Add(1)
	c.wg.Wait()
}
