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
}

func onReqLogin(sender appframe.Session, req *cmsg.CReqLogin) {
	userId := crc64.Checksum([]byte(req.Account), crc64.MakeTable(crc64.ISO))

	sid := sender.ID()
	u, ok := userMgr.findUserBySessionID(sid)
	if !ok {
		u = userMgr.newUser(userId)
		u.account = req.Account
		userMgr.addSession(sid, u)
	}

	u.onConnect(sid)

	u.loginState = StateLogin
	resp := &cmsg.SRespLogin{Code: 0}
	sender.SendMsg(resp)
}

func onReqSendChatMessage(sender appframe.Session, req *cmsg.CReqSendChatMessage) {
	//sid := sender.ID()
	//u, _ := userMgr.findUserBySessionID(sid)
	userMgr.execByEverySession(func(u *user, b bool) {
		if u.session == sender.ID() {
			return
		}
		u.SendMsg(&cmsg.SNotifyUserChatMessage{
			Account: u.account,
			Text:    req.Text,
		})
	})
	resp := &cmsg.SRespSendChatMessage{}
	sender.SendMsg(resp)
}
