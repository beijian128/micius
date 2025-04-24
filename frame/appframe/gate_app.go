package appframe

import (
	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
	"github/beijian128/micius/frame/framework/netcluster"
	"github/beijian128/micius/frame/framework/netframe"
	"github/beijian128/micius/frame/util"
)

// GateApplication 面向客户端连接管理的 Application.
type GateApplication struct {
	Application

	router map[uint32]GateSessionRawMsgRouter
}

// NewGateApplication 创建 Application
// name 参数为 netconfigFile 中对应的 name 值
func NewGateApplication(netconfigFile string, name string) (*GateApplication, error) {
	netconfig, err := netcluster.ParseClusterConfigFile(netconfigFile)
	if err != nil {
		return nil, err
	}
	a := new(GateApplication)
	err = a.init(netconfig, name)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (a *GateApplication) IsShortNetwork() bool {
	return false
}

func (a *GateApplication) init(netconfig *netcluster.ClusterConf, name string) error {
	err := a.Application.init(netconfig, name)
	if err != nil {
		return err
	}
	a.router = make(map[uint32]GateSessionRawMsgRouter)
	// 将来自用户会话的消息转发到目标服务器.
	a.slave.ListenClientBytes(func(sid uint32, _ uint32, msgid uint32, data []byte, extend netframe.Server_Extend) {
		if a.IsExit() {
			return
		}
		router, ok := a.router[msgid]
		if ok {
			router(toSession64(sid), msgid, data, extend.ExtParam)
		} else {
			logrus.WithFields(logrus.Fields{
				"session": sid,
				"msgid":   msgid,
			}).Warn("Unregister msg for route")
		}
	})
	// 将来自服务的消息转发给指定用户会话.
	a.slave.ListenServerBytes(func(_ uint32, _ uint32, msgid uint32, data []byte, extend netframe.Server_Extend) {
		if a.IsExit() {
			return
		}
		a.slave.SendClientBytes(toSession32(extend.SessionId), msgid, data, extend)
	})
	return nil
}

// ListenSessionEvent 监听 session 事件.
func (a *GateApplication) ListenSessionEvent(onNew func(sid uint64), onClose func(sid uint64)) {
	a.slave.ListenClientNetEvent(func(sid uint32, _ uint32) {
		if a.IsExit() {
			return
		}
		onNew(toSession64(sid))
	}, func(sid uint32, _ uint32) {
		if a.IsExit() {
			return
		}
		onClose(toSession64(sid))
	})
}

// ListenSessionMsg 监听来自 session 的消息, 功能相近于 ListenMsg.
func (a *GateApplication) ListenSessionMsg(msg proto.Message, handler GateSessionMsgHandler) {
	isResponseMessage := a.isResponseMessage(msg)
	a.slave.ListenClientMessage(msg, func(sid uint32, _ uint32, _ uint32, _ []byte, imsg any, extend netframe.Server_Extend) {
		if a.IsExit() {
			if !isResponseMessage {
				return
			}
		}
		msg, ok := imsg.(proto.Message)
		if !ok {
			logrus.Error("Msg from session is not a proto.Message")
			return
		}
		session := &gateSession{
			id:     toSession64(sid),
			app:    a,
			extend: extend,
		}
		if a.logicWorker != nil { // 在业务自定义的消息函数执行者中执行回调函数.
			a.logicWorker.Post(func() {
				handler(session, msg)
			})
		} else { // 默认在 app 的 ioWorker 中执行回调函数.
			handler(session, msg)
		}
	})
}

// RouteSessionRawMsg 设置消息路由函数, 未 ListenSessionMsg 的消息会调用该函数.
func (a *GateApplication) RouteSessionRawMsg(msg proto.Message, router GateSessionRawMsgRouter) {
	msgid := util.StringHash(proto.MessageName(msg))
	a.router[msgid] = router
}

// GetSession 获取用户会话对象
// 使用者不用对返回结果判空, 对无效 session 的错误处理, 都会延迟到调用 GateSession 的方法时发生, 这样做的目的就是为了简化错误处理.
func (a *GateApplication) GetSession(sid uint64) GateSession {
	return &gateSession{
		id:  sid,
		app: a,
	}
}
