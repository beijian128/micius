package appframeslb

import (
	"github/beijian128/micius/frame/framework/netcluster"
	"sync/atomic"
)

// WithLoadBalanceSingleton 单节点服务负载均衡策略 (即无策略), 目的是统一抽象.
// 参数 app 可以是 *Application, 也可以是 *GateApplication 对象.
func WithLoadBalanceSingleton(app iGetAvailableServerIDs, svrType ServerType) LoadBalance {
	return &singleton{app: app, typ: svrType}
}

// 单节点服务负载均衡策略实现 (即无策略)
type singleton struct {
	app iGetAvailableServerIDs
	typ ServerType
	_id uint32
}

func (s *singleton) GetServerID(ext uint64) (uint32, bool) {
	id := atomic.LoadUint32(&s._id)
	if id != 0 {
		return id, true
	}
	ids := s.app.GetAvailableServerIDs(s.typ)
	if len(ids) > 0 {
		id = ids[0]
		if atomic.CompareAndSwapUint32(&s._id, 0, id) {
			return id, true
		}
		return s.GetServerID(ext)
	}
	return 0, false
}

func (s *singleton) Type() ServerType {
	return s.typ
}
func (s *singleton) Available() bool {
	return len(s.app.GetAvailableServerIDs(s.typ)) > 0
}
func (s *singleton) OnServerEvent(svrid uint32, event netcluster.SvrEvent) {}
