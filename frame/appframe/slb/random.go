package appframeslb

import (
	"github/beijian128/micius/frame/framework/netcluster"
	"math/rand"
)

// WithLoadBalanceRandom 随机选择服务节点负载均衡策略.
// 参数 app 可以是 *Application, 也可以是 *GateApplication 对象.
func WithLoadBalanceRandom(app iGetAvailableServerIDs, svrType uint32) LoadBalance {
	return &random{app: app, typ: svrType}
}

// 随机选择服务节点负载均衡策略实现.
type random struct {
	app iGetAvailableServerIDs
	typ uint32
}

func (r *random) GetServerID(ext uint64) (uint32, bool) {
	ids := r.app.GetAvailableServerIDs(r.typ)
	l := len(ids)
	if l == 0 {
		return 0, false
	}
	return ids[rand.Intn(l)], true
}
func (r *random) Available() bool {
	return len(r.app.GetAvailableServerIDs(r.typ)) > 0
}
func (r *random) OnServerEvent(svrid uint32, event netcluster.SvrEvent) {}
