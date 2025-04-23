package appframeslb

import (
	"fmt"
	"github/beijian128/micius/frame/framework/netcluster"
	"hash/crc32"
	"sort"
)

//一致性哈希（Consistent Hashing）是一种特殊的哈希算法，常用于分布式系统中以实现数据的均匀分布和高效的负载均衡。
//当在一致性哈希环中增加新的节点时，会发生以下几个主要变化：
//哈希环的重构：新节点会被加入到哈希环中。如果使用了虚拟节点（通常这样做是为了更好的均衡分布），新节点的多个虚拟节点也会被分散地加入到环中。
//数据重映射：一致性哈希环的主要特点是，只有与新节点相邻的部分数据需要被重新映射和迁移。这意味着，只有存储在新节点“顺时针方向”紧邻的旧节点上的部分数据需要迁移到新节点上。
//负载重新分配：随着新节点的加入，数据和请求的负载将在更多的节点之间分配，通常这会导致整体负载更加均匀。
//系统性能变化：新节点的加入可以提高系统的处理能力和存储容量，但在数据迁移过程中可能会暂时影响系统性能。
//当从服务集群中删除一个旧节点时，会发生以下几个关键步骤：
//移除节点的哈希值：在哈希环上，该节点（包括其所有虚拟节点，如果有的话）的哈希值将被移除。这意味着环上这些特定的点不再与该节点关联。
//数据重新分配：在一致性哈希环上，删除节点会导致原本分配给该节点的数据需要被重新分配给其他节点。具体来说，这些数据将移动到哈希环上顺时针方向的下一个节点。
//负载重新平衡：随着旧节点的移除，其承载的工作负载（比如数据存储、请求处理等）需要在剩余的节点间重新平衡。这可能会导致系统负载的短期波动，直到达到新的平衡状态。
//系统性能影响：删除节点可能会暂时影响系统性能，特别是在数据迁移和重新分配期间。必须妥善管理这一过程，以最小化对服务的影响。

// WithLoadBalanceConsistentHash 一致性哈希服务节点负载均衡策略.
func WithLoadBalanceConsistentHash(app iGetAvailableServerIDs, svrType ServerType) LoadBalance {
	return newConsistentHash(app, svrType, 100)
}

// consistentHash 一致性哈希 （参考redis集群的SLB方案：基于一致性哈希环，保证一致性的前提下，又能支持动态增删服务节点）
type consistentHash struct {
	app iGetAvailableServerIDs
	typ ServerType

	circle       map[uint32]uint32   //hash环 key为哈希值 值为存放节点的信息
	sortedHashes []uint32            //已经排序的节点hash切片
	virtualNode  uint32              //虚拟节点个数,用来增加hash的平衡性
	nodeIDs      map[uint32]struct{} //真实节点信息
}

func newConsistentHash(app iGetAvailableServerIDs, ServerType ServerType, virtualNodeNum uint32) *consistentHash {
	c := &consistentHash{}
	c.app = app
	c.typ = ServerType
	c.circle = make(map[uint32]uint32)
	c.virtualNode = virtualNodeNum
	c.nodeIDs = make(map[uint32]struct{})
	return c
}

func (c *consistentHash) generateKey(serverID uint32, index int) string {
	return fmt.Sprintf("%d%d", serverID, index)
}

// 获取hash位置
func (c *consistentHash) hashKey(key string) uint32 {
	if len(key) < 64 {
		var keyByte [64]byte
		copy(keyByte[:], key)
		return crc32.ChecksumIEEE(keyByte[:len(key)])
	}
	return crc32.ChecksumIEEE([]byte(key))
}

func (c *consistentHash) updateSortedHashes() {
	hashes := make([]uint32, 0, len(c.circle))
	for k := range c.circle {
		hashes = append(hashes, k)
	}
	sort.Slice(hashes, func(i, j int) bool {
		return hashes[i] < hashes[j]
	})
	c.sortedHashes = hashes
}

// 增加节点
func (c *consistentHash) addNode(serverID uint32) {
	_, exist := c.nodeIDs[serverID]
	if exist {
		return
	}
	for i := 0; i < int(c.virtualNode); i++ {
		c.circle[c.hashKey(c.generateKey(serverID, i))] = serverID
	}
	c.updateSortedHashes()
	c.nodeIDs[serverID] = struct{}{}
}

// 删除节点
func (c *consistentHash) removeNode(serverID uint32) {
	_, exist := c.nodeIDs[serverID]
	if !exist {
		return
	}
	for i := 0; i < int(c.virtualNode); i++ {
		delete(c.circle, c.hashKey(c.generateKey(serverID, i)))
	}
	c.updateSortedHashes()
	delete(c.nodeIDs, serverID)
}

// 顺时针查找最近的节点
func (c *consistentHash) search(key uint32) int {
	index := sort.Search(len(c.sortedHashes), func(i int) bool {
		return c.sortedHashes[i] > key
	})
	//如果超出范围则设置i=0
	if index >= len(c.sortedHashes) {
		index = 0
	}
	return index
}

func (c *consistentHash) Available() bool {
	return len(c.circle) > 0
}

func (c *consistentHash) GetServerID(ext uint64) (uint32, bool) {
	if len(c.circle) == 0 {
		return 0, false
	}
	i := c.search(c.hashKey(fmt.Sprint(ext)))
	return c.circle[c.sortedHashes[i]], true
}

func (c *consistentHash) OnServerEvent(svrID uint32, event netcluster.SvrEvent) {
	switch event {
	case netcluster.SvrEventStart, netcluster.SvrEventReconnect:
		c.addNode(svrID)
	case netcluster.SvrEventDisconnect, netcluster.SvrEventQuit:
		c.removeNode(svrID)
	}
}

func (c *consistentHash) AddServerNode(serverIDs ...uint32) {
	if len(serverIDs) <= 0 {
		return
	}
	for _, serverID := range serverIDs {
		c.addNode(serverID)
	}
}

func (c *consistentHash) DelServerNode(serverIDs ...uint32) {
	if len(serverIDs) <= 0 {
		return
	}
	for _, serverID := range serverIDs {
		c.removeNode(serverID)
	}
}

func (c *consistentHash) GetCurNodeInfo() map[uint32]struct{} {
	return c.nodeIDs
}
