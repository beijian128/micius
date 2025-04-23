package lobby

import (
	"github/beijian128/micius/frame/appframe"
	"time"

	"github.com/golang/protobuf/proto"
)

type LoginState int8

const (
	// 未登录状态
	StateLogout LoginState = 0
	// 登录中
	StateLogining LoginState = 1
	// 已登录
	StateLogin LoginState = 2
)

type user struct {
	userid         uint64
	session        appframe.SessionID
	loginState     LoginState
	connect        bool
	disconnectTime int64

	mgr *userManager
}

func (p *user) isConnected() bool {
	return p.connect
}

func (p *user) SendMsg(msg proto.Message) {
	if !p.isConnected() {
		return
	}
	AppInstance.GetSession(p.session).SendMsg(msg)
}

func (p *user) IsConnect() bool {
	return p.connect
}

func (p *user) onDisconnect() {
	if !p.connect {
		return
	}

	p.loginState = StateLogout
	p.connect = false
	p.disconnectTime = time.Now().Unix()

	userMgr.removeSession(p.session)
	p.session = appframe.SessionID{}

}

func (p *user) onConnect(sid appframe.SessionID) {
	//if p.connect {
	//	if p.session == sid {
	//		logrus.WithFields(logrus.Fields{
	//			"userid":  p.userid,
	//			"session": sid,
	//		}).Warn("User reconnect with same session")
	//		return
	//	}
	//	p.mgr.app.GetServer(p.session.SvrID).SendMsg(&smsg.LoGaNtfCloseSession{
	//		Session: p.session.ID,
	//		Reason:  gameconf.KickUserOutReason_KUORelogin,
	//		Msg:     "",
	//	})
	//	p.onDisconnect()
	//}

	p.connect = true
	p.session = sid
	p.mgr.addSession(sid, p)

}
