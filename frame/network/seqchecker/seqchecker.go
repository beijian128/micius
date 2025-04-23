package seqchecker

import (
	"encoding/binary"
	"time"
)

const (
	SeqIDSize = 12 // 协议序列号（唯一标识符）长度 为12字节 。 4 字节存储timestamp + 8 字节存储 Nonce（随机数）

	SeqExpireTime = time.Minute * 5
)

type SeqIDChecker struct {
	hSet      hashSet
	byteOrder binary.ByteOrder
}

func NewSeqIDChecker(order binary.ByteOrder) *SeqIDChecker {
	return &SeqIDChecker{
		hSet:      newHashSetImp(),
		byteOrder: order,
	}
}

func (c *SeqIDChecker) GetSeqIDSize() int {
	return SeqIDSize
}

// CanUse 序列号是否可用
func (c *SeqIDChecker) CanUse(seqData []byte) bool {
	//ts := c.byteOrder.ServerType(seqData[:4])
	//noce:=c.byteOrder.Uint64(seqData[4:])
	//now := uint32(time.Now().Unix())
	//if absDeltaTime(now, ts) > SeqExpireTime { // 先比对时间戳，拒绝接收已经过期的消息
	//	logrus.WithFields(logrus.Fields{
	//		"now unix": now,
	//		"ts unix":  ts,
	//	}).Warn("出现已过期的消息")
	//	return false
	//}
	//num := string(seqData)
	//if c.hSet.get(num) { // 缓存中能找到该序列号，说明消息被重放
	//	logrus.Warn("消息被重放")
	//	return false
	//}
	//c.hSet.set(num, SeqExpireTime) // 允许接收消息，并标记此序列号为已使用
	return true
}

func (c *SeqIDChecker) Use(number string) {
	c.hSet.set(number, SeqExpireTime) // 允许接收消息，并标记此序列号为已使用
}

func absDeltaTime(a, b uint32) time.Duration {
	if a > b {
		return time.Duration(a-b) * time.Second
	}
	return time.Duration(b-a) * time.Second
}
