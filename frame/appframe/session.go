package appframe

import "github.com/golang/protobuf/proto"
import "github/beijian128/micius/frame/framework/netframe"

// SessionID 会话标识符
type SessionID struct {
	SvrID uint32 `json:"svrid"`
	ID    uint64 `json:"id"`
}

// Session 用户会话
type Session interface {
	// ID 唯一标识符
	ID() SessionID
	// SendMsg 发送消息给会话
	SendMsg(msg proto.Message) error
}

type svrSession struct {
	id  SessionID
	app *Application
}

func (s *svrSession) ID() SessionID {
	return s.id
}

func (s *svrSession) SendMsg(msg proto.Message) error {
	return s.app.slave.SendServerMsg(msg, netframe.Server_Extend{ServerId: s.id.SvrID, SessionId: s.id.ID})
}

type SessionIDExtend struct {
	SessionID
	UserID   uint64
	ExtParam int64
}

type svrSessionExtend struct {
	id  SessionIDExtend
	app *Application
}

func (s *svrSessionExtend) ID() SessionID {
	return s.id.SessionID
}

func (s *svrSessionExtend) SendMsg(msg proto.Message) error {
	return s.app.slave.SendServerMsg(msg, netframe.Server_Extend{ServerId: s.id.SvrID, SessionId: s.id.ID, UserId: s.id.UserID, ExtParam: s.id.ExtParam})
}

// SessionMsgHandler 用户会话消息回调函数定义
type SessionMsgHandler func(s Session, msg proto.Message)
