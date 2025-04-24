package websocket

import (
	"errors"
	"io"
	"net"
	"sync/atomic"

	"github/beijian128/micius/frame/network/msgpackager"
	"github/beijian128/micius/frame/network/msgprocessor"
	"github/beijian128/micius/frame/util"

	"github.com/gorilla/websocket"
)

var (
	// ErrClosed 已关闭
	ErrClosed = errors.New("ErrClosed")
	// ErrWriteChanFull 写管道满了
	ErrWriteChanFull = errors.New("ErrWriteChanFull")
)

type msg interface {
	marshal() (ext any, msgid uint32, data []byte, err error)
}

type bytesMsg struct {
	ext  any
	id   uint32
	data []byte
}

func (m *bytesMsg) marshal() (any, uint32, []byte, error) {
	return m.ext, m.id, m.data, nil
}

type protoMsg struct {
	ext any
	msg any
}

func (m *protoMsg) marshal() (any, uint32, []byte, error) {
	msgid, data, err := msgprocessor.OnMarshal(m.msg)
	return m.ext, msgid, data, err
}

type connection struct {
	c        *websocket.Conn
	isServer bool

	msgPackager  msgpackager.MsgPackager
	msgProcessor msgprocessor.MsgProcessor

	closed        int32
	writeCh       chan msg
	closeBuffFull bool
}

func newConnection(
	conn *websocket.Conn,
	isServer bool,
	writeChanLen int,
	closeBuffFull bool,
	msgPackager msgpackager.MsgPackager,
	msgProcessor msgprocessor.MsgProcessor,
) *connection {
	c := new(connection)
	c.c = conn
	c.isServer = isServer
	c.msgPackager = msgPackager
	c.msgProcessor = msgProcessor
	c.writeCh = make(chan msg, writeChanLen)
	c.closeBuffFull = closeBuffFull

	go c.writeLoop()
	return c
}

// connection.Connection interface
func (c *connection) Name() string {
	return c.c.LocalAddr().String()
}

func (c *connection) WriteMsg(ext any, msg any) error {
	return c.write(&protoMsg{
		ext: ext,
		msg: msg,
	})
}

func (c *connection) WriteBytes(ext any, msgid uint32, data []byte) error {
	return c.write(&bytesMsg{
		ext:  ext,
		id:   msgid,
		data: data,
	})
}

func (c *connection) Close() {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		close(c.writeCh)
	}
}

// connection.Connection interface
func (c *connection) LocalAddr() net.Addr {
	return c.c.LocalAddr()
}

// connection.Connection interface
func (c *connection) RemoteAddr() net.Addr {
	return c.c.RemoteAddr()
}

func (c *connection) isClosed() bool {
	return atomic.LoadInt32(&c.closed) != 0
}

func (c *connection) write(msg msg) error {
	if c.isClosed() {
		return ErrClosed
	}

	if c.closeBuffFull && len(c.writeCh) == cap(c.writeCh) {
		logger.WithField("name", c.Name()).Warning("conn will close because of channel full")
		c.Close()
		return ErrWriteChanFull
	}

	c.writeCh <- msg

	return nil
}

func (c *connection) writeLoop() {
	logger := logger.WithField("name", c.Name())

	defer func() {
		c.c.Close()
	}()

	for msg := range c.writeCh {
		ext, msgid, msgData, err := msg.marshal()
		if err != nil {
			logger.WithError(err).Error("Marshal message error")
			continue
		}

		var extData []byte
		if ext != nil {
			extData, err = c.msgProcessor.OnEncodeExt(ext)
			if err != nil {
				logger.WithField("msgid", msgid).WithError(err).Error("Encode ext error")
				continue
			}
		}

		w, err := c.c.NextWriter(websocket.BinaryMessage)
		if err != nil {
			logger.WithField("msgid", msgid).WithError(err).Error("Conn write message(NextWriter)")
			break
		}

		err = c.msgPackager.WriteMsg(w, msgid, extData, msgData)
		if err != nil {
			w.Close()
			logger.WithField("msgid", msgid).WithError(err).Error("Conn write message")
			break
		}

		w.Close()
	}

	if c.isServer {
		logger.WithField("name", c.Name()).Debug("Conn exit write loop")
	} else {
		logger.WithField("name", c.Name()).Debug("Conn exit write loop")
	}
}

func (c *connection) readLoop() {
	logger := logger.WithField("name", c.Name())

	defer util.Recover()
	defer func() {
		c.msgProcessor.OnClose(c)
		c.Close()
	}()

	c.msgProcessor.OnConnect(c)

	for {
		if c.isClosed() {
			break
		}

		frameType, r, err := c.c.NextReader()
		if err != nil {
			logger.WithError(err).Warn("New reader failed")
			break
		}
		if frameType != websocket.BinaryMessage {
			logger.Warn("Not a websocket.BinaryMessage")
			break
		}

		msgid, extData, msgData, err := c.msgPackager.ReadMsg(r)
		if c.isClosed() {
			break
		}
		if err != nil {
			if err != io.EOF {
				logger.WithError(err).Error("Conn read message error")
			}
			break
		}

		ext, err := c.msgProcessor.OnDecodeExt(extData)
		if err != nil {
			logger.WithField("msgid", msgid).WithError(err).Error("Decode ext error")
			break
		}

		err = c.msgProcessor.OnMessage(c, ext, msgid, msgData)
		if err != nil {
			logger.WithField("msgid", msgid).WithError(err).Error("OnMessage error (websocket)")
			break
		}
	}

	if c.isServer {
		logger.WithField("name", c.Name()).Trace("Conn exit read loop")
	} else {
		logger.WithField("name", c.Name()).Traceln("Conn exit read loop")
	}
}
