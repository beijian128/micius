package gate

import (
	"github/beijian128/micius/frame/appframe"
	"github/beijian128/micius/proto/smsg"
	"github/beijian128/micius/services"
)

type session struct {
	appframe.GateSession
	userid  uint64
	account string
}

func newSession() *session {
	s := new(session)

	return s
}

type sessionManager struct {
	sid2session map[uint64]*session
	uid2session map[uint64]*session
	uid         uint64
}

func (sm *sessionManager) addSession(gateSession appframe.GateSession) *session {
	s := newSession()
	sm.uid++
	s.userid = sm.uid
	s.GateSession = gateSession

	sm.sid2session[s.ID()] = s

	return s
}
func (sm *sessionManager) removeSession(sid uint64) {
	s, ok := sm.sid2session[sid]
	if ok {
		us, exist := sm.getSessionByUserID(s.userid)
		if exist && us.ID() == sid {
			delete(sm.uid2session, us.userid)
		}
		delete(sm.sid2session, sid)
	}
}

func (sm *sessionManager) execByEveryUser(f func(uid uint64, s *session)) {
	for uid, session := range sm.uid2session {
		f(uid, session)
	}
}

func (sm *sessionManager) getSessionByUserID(userID uint64) (*session, bool) {
	if s, ok := sm.uid2session[userID]; ok {
		return s, true
	}
	return nil, false
}

func (sm *sessionManager) getSession(sid uint64) (*session, bool) {
	s, ok := sm.sid2session[sid]
	return s, ok
}

func (p *sessionManager) execByEverySession(f func(*session)) {
	for _, s := range p.uid2session {
		f(s)
	}
}

func initSessionManager(app *appframe.GateApplication) *sessionManager {
	sm := new(sessionManager)
	sm.sid2session = make(map[uint64]*session)
	sm.uid2session = make(map[uint64]*session)

	// session 的心跳已经由底层维护了.

	// 监听 session 的开启和关闭事件.
	app.ListenSessionEvent(func(sid uint64) {
		sm.addSession(app.GetSession(sid))
	}, func(sid uint64) {
		if _, ok := sm.getSession(sid); ok {
			// 通知 lobby 玩家断开连接.
			app.GetService(services.ServiceType_Lobby).SendMsg(&smsg.NoticeSessionClosed{
				Session: sid,
			})
			sm.removeSession(sid)
		}
	})

	return sm
}
