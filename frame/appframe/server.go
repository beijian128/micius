package appframe

import (
	appframeslb "github/beijian128/micius/frame/appframe/slb"
	"time"

	"github.com/golang/protobuf/proto"
	"github/beijian128/micius/frame/framework/netframe"
)

// Server 一个具体的服务器节点, 有唯一的 ID 号标识.
type Server interface {
	// ID 服务唯一标识符.
	ID() uint32
	// Service 继承自 Service 接口
	Service
}

// MsgHandler 消息处理函数声明
type MsgHandler func(sender Server, extend netframe.Server_Extend, msg proto.Message)

type server struct {
	id  uint32
	app *Application
}

func (s *server) ID() uint32 {
	return s.id
}

func (s *server) Type() uint32 {
	typ, _ := s.app.slave.GetServerType(s.id)
	return uint32(typ)
}

func (s *server) Available() bool {
	return s.app.slave.IsServerAvailable(s.id)
}

func (s *server) SendMsg(msg proto.Message) error {
	return s.app.slave.SendServerMsg(msg, netframe.Server_Extend{ServerId: s.id, UserId: getUserID(msg)})
}

func (s *server) SendMsgExtend(msg proto.Message, extend netframe.Server_Extend) error {
	extend.ServerId = s.id
	extend.UserId = getUserID(msg)
	return s.app.slave.SendServerMsg(msg, extend)
}

func (s *server) Request(msg proto.Message, cbk func(proto.Message, error), timeout time.Duration) (cancel func()) {
	extend := netframe.Server_Extend{
		ServerId: s.id,
		UserId:   getUserID(msg),
	}
	return s.app.reqc.Req(s.SendMsgExtend, msg, extend, cbk, timeout)
}

func (s *server) RequestCall(msg proto.Message, timeout time.Duration) (proto.Message, error) {
	extend := netframe.Server_Extend{
		ServerId: s.id,
		UserId:   getUserID(msg),
	}
	return s.app.reqc.Call(s.SendMsgExtend, msg, extend, timeout)
}

func (s *server) ForwardMsgFromSession(msg proto.Message, extend netframe.Server_Extend) error {
	extend.ServerId = s.id
	return s.app.slave.SendServerMsg(msg, extend)
}

func (s *server) ForwardRawMsgFromSession(msgid uint32, data []byte, extend netframe.Server_Extend) error {
	extend.ServerId = s.id
	return s.app.slave.SendServerBytes(msgid, data, extend)
}
