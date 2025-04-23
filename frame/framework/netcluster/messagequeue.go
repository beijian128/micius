package netcluster

import (
	"github/beijian128/micius/frame/framework/netframe"
)

// wMessage ..
type wMessage struct {
	msg   any
	bytes []byte
	pri   int64
	index int

	extend netframe.Server_Extend

	msgID uint32
}

type msgQueue []*wMessage

func newMsgQueue(capacity int) msgQueue {
	return make(msgQueue, 0, capacity)
}

// Swap a<->b
func (pq msgQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push q<-x
func (pq *msgQueue) Push(x *wMessage) {
	n := len(*pq)
	c := cap(*pq)
	if n+1 > c {
		npq := make(msgQueue, n, c*2)
		copy(npq, *pq)
		*pq = npq
	}
	*pq = (*pq)[0 : n+1]
	x.index = n
	(*pq)[n] = x
	pq.up(n)
}

func (pq *msgQueue) Pop() *wMessage {
	n := len(*pq)
	if n == 0 {
		return nil
	}
	c := cap(*pq)
	pq.Swap(0, n-1)
	pq.down(0, n-1)
	if n < (c/2) && c > 25 {
		npq := make(msgQueue, n, c/2)
		copy(npq, *pq)
		*pq = npq
	}
	x := (*pq)[n-1]
	x.index = -1
	*pq = (*pq)[0 : n-1]
	return x
}

func (pq *msgQueue) Remove(i int) *wMessage {
	n := len(*pq)
	if n-1 != i {
		pq.Swap(i, n-1)
		pq.down(i, n-1)
		pq.up(i)
	}
	x := (*pq)[n-1]
	x.index = -1
	*pq = (*pq)[0 : n-1]
	return x
}

func (pq *msgQueue) PeekAndShift(max int64) (*wMessage, int64) {
	if len(*pq) == 0 {
		return nil, 0
	}

	x := (*pq)[0]
	if x.pri > max {
		return nil, x.pri - max
	}
	pq.Pop()

	return x, 0
}

func (pq *msgQueue) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || (*pq)[j].pri >= (*pq)[i].pri {
			break
		}
		pq.Swap(i, j)
		j = i
	}
}

func (pq *msgQueue) down(i, n int) {
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && (*pq)[j1].pri >= (*pq)[j2].pri {
			j = j2 // = 2*i + 2  // right child
		}
		if (*pq)[j].pri >= (*pq)[i].pri {
			break
		}
		pq.Swap(i, j)
		i = j
	}
}
