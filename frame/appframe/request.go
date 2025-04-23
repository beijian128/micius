package appframe

import (
	"github/beijian128/micius/frame/framework/netframe"
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
	"github/beijian128/micius/frame/appframe/request/protoreq"
)

// Requester 请求者
type Requester interface {
	From() Server
	Resp(proto.Message) error
}

// ListenRequest 监听请求消息, 请求必须有 Seqid int64 字段
func ListenRequest(app iListenMsg, msg proto.Message, handler func(sender Requester, req proto.Message)) {
	app.ListenMsg(msg, func(sender Server, extend netframe.Server_Extend, msg proto.Message) {
		handler(&requester{
			s:      sender,
			extend: extend,
		}, msg)
	})
}

type requester struct {
	s      Server
	extend netframe.Server_Extend
}

func (r *requester) From() Server {
	return r.s
}

// Resp 响应消息必须有 Seqid int64 字段
func (r *requester) Resp(msg proto.Message) error {
	return r.s.SendMsgExtend(msg, r.extend)
}

// NewErrorResponse 创建一个通用的错误响应消息, 该消息会以 error 的形式返回给请求者的回调函数, 帮助简化错误处理
func NewErrorResponse() *protoreq.ErrCode {
	return new(protoreq.ErrCode)
}

// CheckErrorResponse 检查错误是否为 ErrorResponse
func CheckErrorResponse(err error) (*protoreq.ErrCode, bool) {
	ec, ok := err.(*protoreq.ErrCode)
	return ec, ok
}

// ListenRequestSugar 为消息监听处理提供便利.
// app 参数可以是 *Application 或 *GateApplication 对象.
// reqHandler 必须是 func(sender Requester, msg *FooMsg) 形式的函数.
func ListenRequestSugar(app iListenMsg, reqHandler any) {
	v := reflect.ValueOf(reqHandler)

	// type check.
	if v.Type().NumIn() != 2 {
		logrus.Panic("ListenRequestSugar handler params num wrong")
	}
	var tempSender Requester
	if v.Type().In(0) != reflect.TypeOf(&tempSender).Elem() {
		logrus.Panic("ListenRequestSugar handler num in 0 is not Requester")
	}

	msg := reflect.New(v.Type().In(1)).Elem().Interface().(proto.Message)
	ListenRequest(app, msg, func(sender Requester, msg proto.Message) {
		v.Call([]reflect.Value{reflect.ValueOf(sender), reflect.ValueOf(msg)})
	})
}
