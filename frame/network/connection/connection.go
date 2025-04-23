package connection

import (
	"net"
)

// Connection 连接
type Connection interface {
	Name() string

	WriteMsg(ext any, msg any) error
	WriteBytes(ext any, msgid uint32, bytes []byte) error

	Close()
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}
