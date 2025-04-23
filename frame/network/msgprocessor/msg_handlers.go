package msgprocessor

import (
	"reflect"
	"sync"

	"github.com/sirupsen/logrus"
)

// MsgHandlers ...
type MsgHandlers struct {
	handlers map[reflect.Type]MsgHandler
	rw       sync.RWMutex
}

// NewMsgHandlers ...
func NewMsgHandlers() *MsgHandlers {
	m := new(MsgHandlers)
	m.handlers = make(map[reflect.Type]MsgHandler)
	return m
}

// GetMsgHandler ...
func (m *MsgHandlers) GetMsgHandler(typ reflect.Type) (MsgHandler, bool) {
	m.rw.RLock()
	handler, ok := m.handlers[typ]
	m.rw.RUnlock()
	return handler, ok
}

// AddHandler ..
func (m *MsgHandlers) AddHandler(msg any, handler MsgHandler) {
	RegisterMessage(msg)
	msgType := reflect.TypeOf(msg)

	m.rw.Lock()
	defer m.rw.Unlock()

	if _, exist := m.handlers[msgType]; exist {
		logrus.WithField("msgtype", msgType).Warning("MsgHandlers already have handler")
	}

	m.handlers[msgType] = handler
}
