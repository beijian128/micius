package tcp

import (
	"expvar"
	"fmt"
	"github/beijian128/micius/frame/network/seqchecker"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github/beijian128/micius/frame/network/connection"
	"github/beijian128/micius/frame/network/crypto"
	"github/beijian128/micius/frame/network/msgpackager"
	"github/beijian128/micius/frame/network/msgprocessor"
	"github/beijian128/micius/frame/util"
)

var logger = logrus.WithField("module", "network")

const (
	// TCPServerMaxConnCnt 服务器接收最大连接数目
	TCPServerMaxConnCnt = 100000
)

// Server ...
// Tcp服务器类
type Server struct {
	index int

	listener   net.Listener
	listenerWG sync.WaitGroup

	conns      map[net.Conn]*Conn
	connsCnt   *expvar.Int
	connsMutex sync.Mutex
	connsWG    sync.WaitGroup

	// msg packager
	MsgPackager msgpackager.MsgPackager
	// msg msgprocessor
	MsgProcessor msgprocessor.MsgProcessor

	newCrypto func() crypto.Crypto
	*seqchecker.SeqIDChecker
}

// NewTCPServer start no block
func NewTCPServer(name string, addr string, maxConnCnt int, tcpConnWriteChanLen int, bCloseBuffFull bool, msgPackager msgpackager.MsgPackager, msgProcessor msgprocessor.MsgProcessor, newCrypto func() crypto.Crypto, checker *seqchecker.SeqIDChecker) *Server {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.WithField("addr", addr).Error("new tcp server failed")
		return nil
	}

	tcpServer := new(Server)

	tcpServer.conns = make(map[net.Conn]*Conn)
	tempname := fmt.Sprintf("TCPServer-%s's 连接数", name)
	tcpServer.connsCnt = expvar.NewInt(tempname)

	tcpServer.MsgPackager = msgPackager
	tcpServer.MsgProcessor = msgProcessor
	tcpServer.newCrypto = newCrypto
	if tcpServer.newCrypto == nil {
		tcpServer.newCrypto = func() crypto.Crypto { return nil }
	}
	tcpServer.SeqIDChecker = checker

	if maxConnCnt <= 0 {
		maxConnCnt = TCPServerMaxConnCnt
	}

	tcpServer.listener = listener

	go tcpServer.run(name, maxConnCnt, tcpConnWriteChanLen, bCloseBuffFull)

	return tcpServer
}

// Close close socket and block until close all conns
func (server *Server) Close() {
	server.listener.Close()
	server.listenerWG.Wait()

	logger.Debug("tcpserver start close")

	server.connsMutex.Lock()
	for _, conn := range server.conns {
		conn.Close()
	}
	server.connsMutex.Unlock()

	server.connsWG.Wait()

	logger.Debug("tcpserver close succeed")
}

// ForEach ...
func (server *Server) ForEach(f func(conn connection.Connection)) {
	if f == nil {
		return
	}

	server.connsMutex.Lock()
	defer server.connsMutex.Unlock()
	for _, conn := range server.conns {
		f(conn)
	}
}

func (server *Server) run(name string, maxConnCnt int, tcpConnWriteChanLen int, bCloseBuffFull bool) {
	server.listenerWG.Add(1)

	defer util.Recover()
	defer server.listenerWG.Done()

	var tempDelay time.Duration
	for {
		//接受客户端连接
		conn, err := server.listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				logger.WithFields(logrus.Fields{
					"error":         err,
					"retryingDelay": tempDelay,
				}).Warning("accept error")
				time.Sleep(tempDelay)
				continue
			}

			logger.WithField("name", name).Debug("tcpserver exit loop")
			return
		}
		tempDelay = 0

		server.connsMutex.Lock()
		if len(server.conns) >= maxConnCnt {
			server.connsMutex.Unlock()
			conn.Close()
			logger.Warning("too many connections")
			continue
		}

		//新建链接对象
		server.index = server.index + 1
		tcpConn := newTCPConn(fmt.Sprintf("%s-%d", name, server.index), conn, true, tcpConnWriteChanLen, bCloseBuffFull, server.MsgPackager, server.MsgProcessor, server.newCrypto(), server.SeqIDChecker)
		server.conns[conn] = tcpConn
		server.connsCnt.Add(1)
		server.connsMutex.Unlock()
		server.connsWG.Add(1)

		go tcpConn.WriteLoop()

		go func() {
			defer func() {
				tcpConn.Close()

				server.connsMutex.Lock()
				delete(server.conns, conn)
				server.connsCnt.Add(-1)
				server.connsMutex.Unlock()

				server.connsWG.Done()
			}()

			tcpConn.ReadLoop()
		}()
	}
}
