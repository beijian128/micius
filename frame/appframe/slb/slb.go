package appframeslb

import "github/beijian128/micius/frame/framework/netcluster"

// LoadBalance 接口, 用于 Service 负载均衡实现.
type LoadBalance interface {
	Available() bool
	GetServerID(ext uint64) (uint32, bool)
	OnServerEvent(svrID uint32, event netcluster.SvrEvent)
}

// Application 和 GateApplication 都实现了这个接口.
type iGetAvailableServerIDs interface {
	GetAvailableServerIDs(typ uint32) []uint32
	ClusterConfig() *netcluster.ClusterConf
}
