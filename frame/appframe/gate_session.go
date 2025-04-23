package appframe

import (
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	"github/beijian128/micius/frame/framework/netframe"
)

// GateSession 用户会话
type GateSession interface {
	// ID 唯一标识符
	ID() uint64

	// Addr 获取会话地址
	Addr() net.Addr

	// SendMsg 发送消息给会话
	SendMsg(msg proto.Message) error

	// ForwardRawMsg 转发其他服务过来的消息给会话
	ForwardRawMsg(msgid uint32, data []byte) error

	// Close 关闭会话连接
	Close()

	// Session32ID connId
	Session32ID() uint32
}

// GateSessionMsgHandler 用户会话消息回调函数定义
type GateSessionMsgHandler func(session GateSession, msg proto.Message)

// GateSessionRawMsgRouter 转发来自Session的消息的委托函数.
type GateSessionRawMsgRouter func(sid uint64, msgid uint32, data []byte, extParam int64)

type gateSession struct {
	id     uint64
	app    *GateApplication
	extend netframe.Server_Extend
}

func (s *gateSession) ID() uint64 {
	return s.id
}

func (s *gateSession) Addr() net.Addr {
	addr := s.app.slave.RemoteAddr(toSession32(s.id))
	if addr == nil {
		return invalidAddr{}
	}
	return addr
}

func (s *gateSession) SendMsg(msg proto.Message) error {
	if s.extend.ClientData == nil {
		s.extend.ClientData = make([]byte, 12)
	}
	//logrus.Debug("send msg to client:", msg, "ext ", s.extend)
	return s.app.slave.SendClientMsg(toSession32(s.id), msg, s.extend)
}

func (s *gateSession) ForwardRawMsg(msgid uint32, data []byte) error {
	if s.extend.ClientData == nil {
		s.extend.ClientData = make([]byte, 12)
	}
	return s.app.slave.SendClientBytes(toSession32(s.id), msgid, data, s.extend)
}

func (s *gateSession) Close() {
	s.app.slave.Close(toSession32(s.id))
}

func (s *gateSession) Session32ID() uint32 {
	return toSession32(s.id)
}

// 表示一个无效的地址, 用于错误处理.
type invalidAddr struct{}

func (a invalidAddr) Network() string { return "" }
func (a invalidAddr) String() string  { return "invalid" }

// session id 的生成必须保证同一服务内唯一(即使该服务重启), 因此在 session id 的前32位加入了带时间信息的扩展头

var extSessionID = (func() uint64 {
	now := time.Now()
	secondsOfYear := uint32(now.Sub(time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())) / time.Second)
	return uint64(secondsOfYear) << 32
})()

func toSession64(sid uint32) uint64 {
	return extSessionID | uint64(sid)
}

func toSession32(sid uint64) uint32 {
	return uint32(sid & (^extSessionID))
}
