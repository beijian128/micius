package appframeslb

import (
	"github/beijian128/micius/frame/framework/netcluster"
	"math/rand"
)

func WithFixedMod(app iGetAvailableServerIDs, svrType ServerType) LoadBalance {
	return &fixedMod{
		app:        app,
		typ:        svrType,
		fixedCount: uint64(app.ClusterConfig().GetServiceSize(uint32(svrType))),
	}
}

// userId取模后作为索引序号，固定对应到服务列表的服务ID
// e.g.  服务列表 list [501,502,503,504]  ,userId=123 ，index=123%4=3 , svrId = list[3] = 504

type fixedMod struct {
	app        iGetAvailableServerIDs
	typ        ServerType
	fixedCount uint64
}

func (r *fixedMod) Available() bool {
	return len(r.app.GetAvailableServerIDs(r.typ)) > 0
}
func (r *fixedMod) OnServerEvent(svrid uint32, event netcluster.SvrEvent) {}

// QuickSelect 实现快速选择算法
func QuickSelect(arr []uint32, k int) uint32 {
	if len(arr) == 1 {
		return arr[0]
	}

	pivot := arr[rand.Intn(len(arr))]

	lows, highs, pivots := partition(arr, pivot)

	switch {
	case k < len(lows):
		return QuickSelect(lows, k)
	case k < len(lows)+len(pivots):
		return pivots[0]
	default:
		return QuickSelect(highs, k-len(lows)-len(pivots))
	}
}

// partition 分区函数
func partition(arr []uint32, pivot uint32) (lows, highs, pivots []uint32) {
	for _, v := range arr {
		switch {
		case v < pivot:
			lows = append(lows, v)
		case v == pivot:
			pivots = append(pivots, v)
		case v > pivot:
			highs = append(highs, v)
		}
	}
	return lows, highs, pivots
}

func (r *fixedMod) GetServerID(ext uint64) (uint32, bool) {
	if r.fixedCount <= 0 {
		return 0, false
	}
	targetId := int(ext % uint64(r.fixedCount))
	ids := r.app.GetAvailableServerIDs(r.typ) // 注意 ids 是乱序的，每次查询得到的ids顺序都可能是不一致的
	if targetId >= len(ids) {
		return 0, false
	}

	// 使用快速选择算法找到第k小的元素
	selectedID := QuickSelect(ids, targetId)

	return selectedID, true
}
