package appframe

import (
	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
	"github/beijian128/micius/frame/framework/netframe"
	"reflect"
)

// Application 和 GateApplication 都实现了这个接口.
type iListenMsg interface {
	ListenMsg(msg proto.Message, handler MsgHandler)
}

// Application 和 GateApplication 都实现了这个接口.
type iListenConsistentMsg interface {
	ListenMsg(msg proto.Message, handler MsgHandler)
}

// ListenMsgSugar 为消息监听处理提供便利.
// 参数 app 可以是 *Application, 也可以是 *GateApplication 对象.
// msgHandler 必须是 func(sender Server, msg *FooMsg) 形式的函数.
func ListenMsgSugar(app iListenMsg, msgHandler any) {
	v := reflect.ValueOf(msgHandler)

	// type check.
	if v.Type().NumIn() != 2 {
		logrus.Panic("MsgHandleSugar handler params num wrong")
	}
	var tempSender Server
	if v.Type().In(0) != reflect.TypeOf(&tempSender).Elem() {
		logrus.Panic("MsgHandleSugar handler num in 0 is not Server")
	}

	msg := reflect.New(v.Type().In(1)).Elem().Interface().(proto.Message)
	app.ListenMsg(msg, func(sender Server, extend netframe.Server_Extend, msg proto.Message) {
		v.Call([]reflect.Value{reflect.ValueOf(sender), reflect.ValueOf(msg)})
	})
}

// Application 实现了该接口
type iListenSessionMsg interface {
	ListenSessionMsg(msg proto.Message, handler SessionMsgHandler)
}

// ListenSessionMsgSugar 为消息监听处理提供便利.
// 参数 app 可以是 *Application.
// msgHandler 必须是 func(sender Server, msg *FooMsg) 形式的函数.
func ListenSessionMsgSugar(app iListenSessionMsg, msgHandler any) {
	v := reflect.ValueOf(msgHandler)

	// type check.
	if v.Type().NumIn() != 2 {
		logrus.Panic("ListenSessionMsgSugar handler params num wrong")
	}
	var tempSender Session
	if v.Type().In(0) != reflect.TypeOf(&tempSender).Elem() {
		logrus.Panic("ListenSessionMsgSugar handler num in 0 is not Session")
	}
	msg := reflect.New(v.Type().In(1)).Elem().Interface().(proto.Message)
	app.ListenSessionMsg(msg, func(sender Session, msg proto.Message) {
		v.Call([]reflect.Value{reflect.ValueOf(sender), reflect.ValueOf(msg)})
	})
}

// GateApplication 实现了该接口.
type iListenGateSessionMsg interface {
	ListenSessionMsg(msg proto.Message, handler GateSessionMsgHandler)
}

// ListenGateSessionMsgSugar 为消息监听处理提供便利.
// app 参数可以是 *GateApplication 对象.
// msgHandler 必须是 func(sender GateSession, msg *FooMsg) 形式的函数.
func ListenGateSessionMsgSugar(app iListenGateSessionMsg, msgHandler any) {
	v := reflect.ValueOf(msgHandler)

	// type check.
	if v.Type().NumIn() != 2 {
		logrus.Panic("ListenGateSessionMsgSugar handler params num wrong")
	}
	var tempSession GateSession
	if v.Type().In(0) != reflect.TypeOf(&tempSession).Elem() {
		logrus.Panic("ListenGateSessionMsgSugar handler num in 0 is not GateSession")
	}

	msg := reflect.New(v.Type().In(1)).Elem().Interface().(proto.Message)
	app.ListenSessionMsg(msg, func(sender GateSession, msg proto.Message) {
		v.Call([]reflect.Value{reflect.ValueOf(sender), reflect.ValueOf(msg)})
	})
}
