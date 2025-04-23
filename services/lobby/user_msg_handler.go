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
	u, ok := userMgr.findUser(userId)
	if !ok {
		u = userMgr.addUser(userId)
	}

	u.onConnect(sid)

	u.loginState = StateLogin

}
