package netcluster

// SvrEvent 服务事件
type SvrEvent int

const (
	// SvrEventStart 新服务上线
	SvrEventStart SvrEvent = 1
	// SvrEventQuit 服务退出
	SvrEventQuit SvrEvent = 2

	// 服务断线状态本质上应该是一种服务质量报告,
	// 但考虑到业务层很可能会有一些重要的状态数据需要在恢复连接时进行同步, 因此还是将其定义为基础事件.

	// SvrEventDisconnect 服务断线
	SvrEventDisconnect SvrEvent = 3
	// SvrEventReconnect 服务重连
	SvrEventReconnect SvrEvent = 4
)

// SvrEventHandler ...
type SvrEventHandler func(svrid uint32, event SvrEvent)
