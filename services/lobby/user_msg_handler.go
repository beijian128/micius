package lobby

import (
	"github/beijian128/micius/frame/appframe"
	"github/beijian128/micius/proto/cmsg"
	"github/beijian128/micius/proto/smsg"
	"hash/crc64"
)

func initUserMsgHandler(app *appframe.Application) {

	appframe.ListenMsgSugar(app, onNoticeSessionClosed)

	appframe.ListenSessionMsgSugar(app, onReqLogin)
	appframe.ListenSessionMsgSugar(app, onReqSendChatMessage)

}

// 断线事件
func onNoticeSessionClosed(sender appframe.Server, notice *smsg.NoticeSessionClosed) {
	sid := appframe.SessionID{SvrID: sender.ID(), ID: notice.Session}
	u, ok := userMgr.findUserBySessionID(sid)
	if !ok {
		return
	}
	u.onDisconnect()
	userMgr.execByEverySession(func(u2 *user, b bool) {
		if u2.session == sid {
			return
		}
		u2.SendMsg(&cmsg.SNotifyUserLeave{Account: u.account})
	})
}

func onReqLogin(sender appframe.Session, req *cmsg.CReqLogin) {
	userId := crc64.Checksum([]byte(req.Account), crc64.MakeTable(crc64.ISO))

	sid := sender.ID()
	u, ok := userMgr.findUserBySessionID(sid)
	if !ok {
		u = userMgr.newUser(userId)
		userMgr.addSession(sid, u)
	}
	u.account = req.Account

	u.onConnect(sid)

	u.loginState = StateLogin
	resp := &cmsg.SRespLogin{Code: 0}
	sender.SendMsg(resp)

	userMgr.execByEverySession(func(u2 *user, b bool) {
		if u2.session == sender.ID() {
			return
		}
		u2.SendMsg(&cmsg.SNotifyUserEnter{Account: u.account})
	})
}

func onReqSendChatMessage(sender appframe.Session, req *cmsg.CReqSendChatMessage) {
	sid := sender.ID()
	u, _ := userMgr.findUserBySessionID(sid)
	userMgr.execByEverySession(func(u2 *user, b bool) {
		if u2.session == sender.ID() {
			return
		}
		u2.SendMsg(&cmsg.SNotifyUserChatMessage{
			Account: u.account,
			Text:    req.Text,
		})
	})
	resp := &cmsg.SRespSendChatMessage{}
	sender.SendMsg(resp)
}
