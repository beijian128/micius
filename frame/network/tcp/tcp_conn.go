package tcp

import (
	"errors"
	"fmt"
	"github/beijian128/micius/frame/network/seqchecker"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github/beijian128/micius/frame/network/crypto"
	"github/beijian128/micius/frame/network/msgpackager"
	"github/beijian128/micius/frame/network/msgprocessor"
	"github/beijian128/micius/frame/util"
)

var (
	writeMsgChanTimeout   = time.Second * 3
	tcpConnDeadlineSecond = time.Second * 1
	tcpConnWaitSecond     = 3
)

var (
	// ErrClosed 已关闭
	ErrClosed = errors.New("ErrClosed")
	// ErrWriteChanFull 写管道满了
	ErrWriteChanFull = errors.New("ErrWriteChanFull")
	// ErrMsgIsNil 空消息
	ErrMsgIsNil = errors.New("ErrMsgIsNil")
)

type writemsg struct {
	ext   any
	msgid uint32
	msg   any
}

// Conn ...
// 连接到服务器的链接信息
type Conn struct {
	name     string
	conn     net.Conn
	isServer bool

	writeChan           chan *writemsg
	closed              int32
	isCloseWhenBuffFull bool

	msgPackager  msgpackager.MsgPackager
	msgProcessor msgprocessor.MsgProcessor
	crypto       crypto.Crypto
	seqChecker   *seqchecker.SeqIDChecker

	localAddr  net.Addr
	remoteAddr net.Addr
}

func newTCPConn(name string, conn net.Conn, isServer bool, tcpConnWriteChanLen int, bCloseBuffFull bool, msgPackager msgpackager.MsgPackager, msgProcessor msgprocessor.MsgProcessor, crypto crypto.Crypto, checker *seqchecker.SeqIDChecker) *Conn {
	tcpConn := new(Conn)

	tcpConn.name = name
	tcpConn.conn = conn
	tcpConn.isServer = isServer

	tcpConn.writeChan = make(chan *writemsg, tcpConnWriteChanLen)
	tcpConn.isCloseWhenBuffFull = bCloseBuffFull

	tcpConn.msgPackager = msgPackager
	tcpConn.msgProcessor = msgProcessor
	tcpConn.crypto = crypto
	tcpConn.seqChecker = checker

	tcpConn.localAddr = conn.LocalAddr()
	tcpConn.remoteAddr = conn.RemoteAddr()

	return tcpConn
}

func (tcpConn *Conn) log(args ...any) {
	if tcpConn.isServer {
		logrus.WithField("conn", tcpConn.Name()).Traceln(args...)
	} else {
		logrus.WithField("conn", tcpConn.Name()).Info(args...)
	}
}

func (tcpConn *Conn) logError(err error, args ...any) {
	logrus.WithField("conn", tcpConn.Name()).WithError(err).Error(args...)
}

func (tcpConn *Conn) isClosed() bool {
	return atomic.LoadInt32(&tcpConn.closed) != 0
}

func (tcpConn *Conn) close() bool {
	if atomic.CompareAndSwapInt32(&tcpConn.closed, 0, 1) {
		tcpConn.log("Conn close succeed")
		return true
	}
	return false
}

func (tcpConn *Conn) closeConn(waitSec int) {
	conn, ok := tcpConn.conn.(*net.TCPConn)
	if ok {
		now := time.Now()
		_ = conn.SetReadDeadline(now.Add(tcpConnDeadlineSecond))
		_ = conn.SetWriteDeadline(now.Add(tcpConnDeadlineSecond))
		_ = conn.SetLinger(waitSec)
		err := conn.Close()
		if err != nil {
			tcpConn.logError(err, "Conn closeConn error")
		} else {
			tcpConn.log("Conn closeConn succeed")
		}
	}
}

// Close ...
func (tcpConn *Conn) Close() {
	if tcpConn.isClosed() {
		return
	}
	tcpConn.log("Conn Close start")
	err := tcpConn.doWrite(nil, 0, nil)
	if err == ErrWriteChanFull {
		tcpConn.log("Conn Close start force close conn")
		tcpConn.closeConn(0)
	}
	if tcpConn.close() {
		tcpConn.log("Conn Close start succeed")
	}
}

func (tcpConn *Conn) postWriteMsg(msg *writemsg) bool {
	select {
	case tcpConn.writeChan <- msg:
		return true
	default:
		return false
	}
}

func (tcpConn *Conn) doWrite(ext any, msgid uint32, msg any) error {
	// 需要控制的话，使用select+timeout 方式
	if tcpConn.isCloseWhenBuffFull && len(tcpConn.writeChan) == cap(tcpConn.writeChan) {
		tcpConn.logError(ErrWriteChanFull, "Conn doWrite error")
		//
		// 异常就销毁这个socket
		//
		tcpConn.closeConn(0)
		if tcpConn.close() {
			close(tcpConn.writeChan)
		}
		return ErrWriteChanFull
	}

	sendmsg := &writemsg{ext, msgid, msg}
	//tcpConn.writeChan <- sendmsg
	ok := tcpConn.postWriteMsg(sendmsg)
	if !ok {
		return ErrWriteChanFull
	}
	return nil
}

// Name ...
func (tcpConn *Conn) Name() string {
	return tcpConn.name
}

// WriteMsg ...
func (tcpConn *Conn) WriteMsg(ext any, msg any) error {
	if msg == nil {
		return ErrMsgIsNil
	}
	if tcpConn.isClosed() {
		return ErrClosed
	}
	return tcpConn.doWrite(ext, 0, msg)
}

// WriteBytes ...
func (tcpConn *Conn) WriteBytes(ext any, msgid uint32, bytes []byte) error {
	if tcpConn.isClosed() {
		return ErrClosed
	}
	return tcpConn.doWrite(ext, msgid, bytes)
}

// LocalAddr ...
func (tcpConn *Conn) LocalAddr() net.Addr {
	return tcpConn.localAddr
}

// RemoteAddr ...
func (tcpConn *Conn) RemoteAddr() net.Addr {
	return tcpConn.remoteAddr
}

// WriteLoop ...
func (tcpConn *Conn) WriteLoop() {
	defer util.Recover()
	defer func() {
		tcpConn.closeConn(tcpConnWaitSecond)
	}()

	for msg := range tcpConn.writeChan {
		if msg == nil || msg.msg == nil {
			tcpConn.log("Conn WriteLoop exit loop")
			break
		}

		var msgid uint32
		var data []byte

		if msg.msgid == 0 {
			var err1 error
			// 组织消息
			msgid, data, err1 = msgprocessor.OnMarshal(msg.msg)
			if err1 != nil {
				if tcpConn.isServer {
					tcpConn.log(err1, "Marshal message error, msgid: ", msgid)
				} else {
					tcpConn.logError(err1, "Marshal message error, msgid: ", msgid)
				}
				continue
			}
		} else {
			// byteSliceTypeName = reflect.TypeOf([]byte(nil))
			// 一定得是[]byte类型
			msgid = msg.msgid
			data = msg.msg.([]byte)
		}

		// 加密消息
		extData, err2 := tcpConn.msgProcessor.OnEncodeExt(msg.ext)
		if err2 != nil {
			tcpConn.logError(err2, "Encode message error, msgid: ", msgid)
			break
		}

		// 打包消息
		err3 := tcpConn.msgPackager.WriteMsg(tcpConn.conn, msgid, extData, data, tcpConn.crypto)
		if err3 != nil {
			if err3 != io.EOF {
				if tcpConn.isServer {
					tcpConn.log(err3, "Conn write message, msgid: ", msgid)
				} else {
					tcpConn.logError(err3, "Conn write message, msgid: ", msgid)
				}
			}
			continue
		}
	}

	tcpConn.log(tcpConn.Name(), " Conn exit write loop")
}

// ReadLoop ...
func (tcpConn *Conn) ReadLoop() {
	defer util.Recover()
	defer func() {
		tcpConn.msgProcessor.OnClose(tcpConn)
		tcpConn.Close()
	}()

	tcpConn.msgProcessor.OnConnect(tcpConn)

	for {
		msgid, extData, msgData, err1 := tcpConn.msgPackager.ReadMsg(tcpConn.conn, tcpConn.crypto, tcpConn.seqChecker)
		if err1 != nil {
			if err1 != io.EOF {
				if tcpConn.isServer {
					logger.WithFields(logrus.Fields{
						"name": tcpConn.Name(),
					}).WithError(err1).Trace("Conn read message error")
				} else {
					logger.WithFields(logrus.Fields{
						"name": tcpConn.Name(),
					}).WithError(err1).Trace("Conn read message error")
				}
			}
			break
		}

		ext, err2 := tcpConn.msgProcessor.OnDecodeExt(extData)
		if err2 != nil {
			logger.WithFields(logrus.Fields{
				"conn":  fmt.Sprintf("%s(%s)", tcpConn.Name(), tcpConn.RemoteAddr().String()),
				"msgid": msgid,
			}).WithError(err2).Error("Decode ext error")
			break
		}

		err3 := tcpConn.msgProcessor.OnMessage(tcpConn, ext, msgid, msgData)
		if err3 != nil {
			logger.WithFields(logrus.Fields{
				"conn":  tcpConn.Name(),
				"msgid": msgid,
			}).WithError(err3).Error("OnMessage error")

			//消息处理出错。这里不断开连接
			continue
		}
	}

	tcpConn.log(tcpConn.Name(), "Conn exit read loop")
}
