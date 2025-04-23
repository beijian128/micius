package msgprocessor

import (
	"fmt"
	"reflect"

	"github.com/golang/protobuf/proto"

	"github/beijian128/micius/frame/ioservice"
	"github/beijian128/micius/frame/network/connection"
)

// MetaProcessor ...
type MetaProcessor struct {
	MsgProcessor
	ConnectHandler ConnectHandler
	CloseHandler   CloseHandler
	MsgHandlers    GetMsgHandler
	BytesHandler   BytesHandler

	//MonitorHandler   func(conn connection.Connection,ext any,msgId uint32,msgData []byte)

	extType    reflect.Type
	callbackIO worker.Worker
}

// NewMetaProcessor ...
func NewMetaProcessor(ext any, worker worker.Worker) *MetaProcessor {
	if worker == nil {
		panic("init MetaProcessor ioservice is nil")
	}
	extType := reflect.TypeOf(ext)
	if extType != nil && (extType.Kind() != reflect.Ptr || extType.Elem() == nil) {
		panic("init MetaProcessor message ext type err, required pointer")
	}

	p := new(MetaProcessor)
	p.extType = extType
	p.callbackIO = worker
	return p
}

// OnDecodeExt decode扩展数据
func (p *MetaProcessor) OnDecodeExt(extData []byte) (ext any, err error) {
	if extData != nil && p.extType != nil {
		ext = reflect.New(p.extType.Elem()).Interface()
		err = proto.UnmarshalMerge(extData, ext.(proto.Message))
		return ext, err
	}
	return extData, err
}

// OnEncodeExt encode扩展数据
func (p *MetaProcessor) OnEncodeExt(ext any) (extData []byte, err error) {
	if ext == nil {
		return nil, nil
	}
	if ext, ok := ext.([]byte); ok {
		return ext, nil
	}
	if ext, ok := ext.(proto.Message); ok {
		return proto.Marshal(ext)
	}
	return nil, fmt.Errorf("OnEncodeExt ext %s not a proto.Message type", reflect.TypeOf(ext))
}

// OnConnect ...
func (p *MetaProcessor) OnConnect(conn connection.Connection) {
	if p.ConnectHandler != nil {
		p.callbackIO.Push(func() {
			p.ConnectHandler(conn)
		})
	}
}

// OnClose ...
func (p *MetaProcessor) OnClose(conn connection.Connection) {
	if p.CloseHandler != nil {
		p.callbackIO.Push(func() {
			p.CloseHandler(conn)
		})
	}
}

// OnMessage 消息回调
func (p *MetaProcessor) OnMessage(conn connection.Connection, ext any, msgid uint32, msgData []byte) error {
	handler, ok := p.findMsgHandler(msgid)
	if !ok {
		if p.BytesHandler != nil {
			p.callbackIO.Push(func() {
				p.BytesHandler(conn, ext, msgid, msgData)
			})
		}
		return nil
	}

	msg, err := OnUnmarshal(msgid, msgData)
	if err != nil {
		return err
	}

	p.callbackIO.Push(func() {
		handler(conn, ext, msgid, msgData, msg)
	})

	return nil
}

func (p *MetaProcessor) findMsgHandler(msgid uint32) (MsgHandler, bool) {
	if p.MsgHandlers == nil {
		return nil, false
	}

	typ, ok := MessageType(msgid)
	if !ok {
		return nil, false
	}

	return p.MsgHandlers.GetMsgHandler(typ)
}
