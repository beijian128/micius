package msgpackager

import (
	"encoding/binary"
	"github/beijian128/micius/frame/network/crypto"
	"github/beijian128/micius/frame/network/seqchecker"
	"io"
)

// MsgPackager 管理协议的组织
type MsgPackager interface {
	ReadMsg(reader io.Reader, crypto crypto.Crypto, checker *seqchecker.SeqIDChecker) (msgId uint32, extData []byte, msgData []byte, error error)
	WriteMsg(writer io.Writer, id uint32, extdata []byte, msgdata []byte, crypto crypto.Crypto) error
}

var (
	// BigEndian ...
	BigEndian = binary.ByteOrder(binary.BigEndian)
	// LittleEndian ...
	LittleEndian = binary.ByteOrder(binary.LittleEndian)
)

const (
	MessageIDSize = 4 // MessageIDSize  4个字节长度

	MessageLenSize = 2 // MessageLenSize 消息头中表示消息长度的字节的大小

	MessageMaxLen = 102400 // MessageMaxLen 消息最大长度

)
