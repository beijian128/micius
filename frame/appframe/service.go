package appframe

import (
	"errors"
	appframeslb "github/beijian128/micius/frame/appframe/slb"
	"time"

	"github.com/golang/protobuf/proto"
	"github/beijian128/micius/frame/framework/netframe"
)

// Service 服务通信接口.
// 一个 Service 对应同一类型的一组服务节点, Service 的具体实现负责消息路由和负载平衡.
type Service interface {
	// Type 服务类型
	Type() uint32

	// Available 服务当前是否可用 (对一个节点来说, 就是没有退出也没有断线)
	Available() bool

	// SendMsg 给服务发送消息.
	SendMsg(msg proto.Message) error

	SendMsgExtend(msg proto.Message, extend netframe.Server_Extend) error

	// Request 对服务进行请求, 请求消息中必须有成员字段 Seqid int64.
	// 异步回调, 回调函数将会在 app 的 ioWorker 中执行.
	// 返回值用于取消等待当前请求的响应.
	Request(msg proto.Message, cbk func(resp proto.Message, err error), timeout time.Duration) (cancel func())

	// RequestCall 对服务进行请求调用, 同步阻塞, 请求消息中必须有成员字段 Seqid int64.
	RequestCall(msg proto.Message, timeout time.Duration) (proto.Message, error)

	// ForwardRawMsgFromSession 转发来自用户会话的原始消息给该服务
	ForwardRawMsgFromSession(msgid uint32, data []byte, extend netframe.Server_Extend) error
}

var (
	// ErrUnregisteredService 未注册的服务.
	ErrUnregisteredService = errors.New("ErrUnregisteredService")
	// ErrNoAvailableServer 没有可用的服务节点.
	ErrNoAvailableServer   = errors.New("ErrNoAvailableServer")
	ErrInvalidProtoMessage = errors.New("ErrInvalidProtoMessage")
)

// unregisteredService 未注册的服务, 用于简化错误处理.
type unregisteredService struct {
	app *Application
}

func (u *unregisteredService) Type() uint32 {
	return 0
}

func (u *unregisteredService) Available() bool {
	return false
}

func (u *unregisteredService) SendMsg(msg proto.Message) error {
	return ErrUnregisteredService
}

func (u *unregisteredService) SendMsgExtend(msg proto.Message, extend netframe.Server_Extend) error {
	return ErrUnregisteredService
}

func (u *unregisteredService) Request(msg proto.Message, cbk func(resp proto.Message, err error), timeout time.Duration) (cancel func()) {
	return func() {

	}
}

func (u *unregisteredService) RequestCall(msg proto.Message, timeout time.Duration) (proto.Message, error) {
	return nil, ErrUnregisteredService
}

func (u *unregisteredService) ForwardRawMsgFromSession(msgid uint32, data []byte, extend netframe.Server_Extend) error {
	return ErrUnregisteredService
}

type GetUserID interface {
	GetUserid() uint64
}

func getUserID(msg proto.Message) uint64 {
	if v, ok := msg.(GetUserID); ok {
		return v.GetUserid()
	}
	return 0
}

// 通用的服务基础结构.
type commonService struct {
	typ         uint32
	app         *Application
	loadBalance appframeslb.LoadBalance
}

func (s *commonService) SendMsg(msg proto.Message) error {
	consistentID := getUserID(msg)
	svrid, ok := s.loadBalance.GetServerID(consistentID)
	if !ok {
		return ErrNoAvailableServer
	}
	return s.app.slave.SendServerMsg(msg, netframe.Server_Extend{ServerId: svrid, UserId: consistentID})
}

func (s *commonService) SendMsgExtend(msg proto.Message, extend netframe.Server_Extend) error {
	svrid, ok := s.loadBalance.GetServerID(extend.UserId)
	if !ok {
		return ErrNoAvailableServer
	}
	extend.ServerId = svrid
	return s.app.slave.SendServerMsg(msg, extend)
}

func (s *commonService) Request(msg proto.Message, cbk func(resp proto.Message, err error), timeout time.Duration) (cancel func()) {
	extend := netframe.Server_Extend{
		UserId: getUserID(msg),
	}
	return s.app.reqc.Req(s.SendMsgExtend, msg, extend, cbk, timeout)
}

func (s *commonService) RequestCall(msg proto.Message, timeout time.Duration) (proto.Message, error) {
	extend := netframe.Server_Extend{
		UserId: getUserID(msg),
	}
	return s.app.reqc.Call(s.SendMsgExtend, msg, extend, timeout)
}

func (s *commonService) Type() uint32 {
	return s.typ
}
func (s *commonService) Available() bool {
	return s.loadBalance.Available()
}

func (s *commonService) ForwardRawMsgFromSession(msgID uint32, data []byte, extend netframe.Server_Extend) error {
	svrID, ok := s.loadBalance.GetServerID(extend.UserId)
	if !ok {
		return ErrNoAvailableServer
	}
	extend.ServerId = svrID
	return s.app.slave.SendServerBytes(msgID, data, extend)
}
