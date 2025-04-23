package lobby

import (
	"github/beijian128/micius/frame/appframe"
)

type userManager struct {
	app     *appframe.Application
	id2user map[uint64]*user
	ss2user map[appframe.SessionID]*user
}

func initUserManager(app *appframe.Application) *userManager {
	userMgr := new(userManager)
	userMgr.app = app
	userMgr.id2user = make(map[uint64]*user)
	userMgr.ss2user = make(map[appframe.SessionID]*user)

	return userMgr
}

func (m *userManager) newUser(userid uint64) *user {
	u := new(user)
	u.userid = userid
	u.mgr = m
	m.id2user[userid] = u
	return u
}

func (m *userManager) addUser(userId uint64) *user {
	u := new(user)
	u.userid = userId
	u.mgr = m
	u.loginState = StateLogout
	m.id2user[userId] = u
	return u
}

func (m *userManager) removeUser(userid uint64) {
	if _, ok := m.id2user[userid]; ok { //u
		delete(m.id2user, userid)
	}
}

func (m *userManager) addSession(sid appframe.SessionID, u *user) {
	u.session = sid
	m.ss2user[sid] = u
}

func (m *userManager) removeSession(sid appframe.SessionID) {
	delete(m.ss2user, sid)
}

func (m *userManager) findUser(userid uint64) (*user, bool) {
	u, ok := m.id2user[userid]
	return u, ok
}

func (m *userManager) findUserBySessionID(sid appframe.SessionID) (*user, bool) {
	u, ok := m.ss2user[sid]
	return u, ok
}

func (m *userManager) execByEverySession(callback func(*user, bool)) {
	if m.ss2user == nil || callback == nil {
		return
	}

	length := len(m.ss2user)
	count := 0
	for _, v := range m.ss2user {
		count++
		callback(v, length == count)
	}
}
