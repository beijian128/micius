package tcp

import (
	"fmt"
	"github/beijian128/micius/frame/network/seqchecker"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github/beijian128/micius/frame/network/connection"

	"github/beijian128/micius/frame/network/msgpackager"
	"github/beijian128/micius/frame/network/msgprocessor"
	"github/beijian128/micius/frame/util"
)

const (
	// TCPClientMaxConnCnt 一个客户端最大连接数
	TCPClientMaxConnCnt = 10000
)

// Client ...
// Tcp客户端类
type Client struct {
	index int

	isClosed bool

	conns map[net.Conn]*Conn
	//connsCnt   *expvar.Int
	connsMutex sync.Mutex
	connsWG    sync.WaitGroup

	// msg packager
	MsgPackager msgpackager.MsgPackager
	// msg msgprocessor
	MsgProcessor msgprocessor.MsgProcessor

	seqChecker *seqchecker.SeqIDChecker
}

// NewTCPClient ...
func NewTCPClient(name string, addr string, connCnt int, isAutoReconnect bool, autoReconnectInterval time.Duration, tcpConnWriteChanLen int, msgPackager msgpackager.MsgPackager, msgProcessor msgprocessor.MsgProcessor) *Client {
	tcpClient := new(Client)

	tcpClient.isClosed = false

	tcpClient.conns = make(map[net.Conn]*Conn)
	// tempname := fmt.Sprintf("TCPClient:%s's 连接数", name)
	// tcpClient.connsCnt = expvar.NewInt(tempname)

	tcpClient.MsgPackager = msgPackager
	tcpClient.MsgProcessor = msgProcessor

	if connCnt > TCPClientMaxConnCnt {
		connCnt = TCPClientMaxConnCnt
	} else if connCnt <= 0 {
		connCnt = 1
	}

	for index := 0; index < connCnt; index++ {
		tcpClient.connsWG.Add(1)
		go tcpClient.run(name, addr, isAutoReconnect, autoReconnectInterval, tcpConnWriteChanLen)
	}

	return tcpClient
}

// Close ...
func (client *Client) Close() {
	logger.Debug("tcpclient start close")

	client.connsMutex.Lock()
	if !client.isClosed {
		client.isClosed = true
		for _, conn := range client.conns {
			conn.Close()
		}
	}
	client.connsMutex.Unlock()

	client.connsWG.Wait()

	logger.Debug("tcpclient close succeed")
}

// ForEach ...
func (client *Client) ForEach(f func(conn connection.Connection)) {
	if f == nil {
		return
	}

	client.connsMutex.Lock()
	defer client.connsMutex.Unlock()
	if !client.isClosed {
		for _, conn := range client.conns {
			f(conn)
		}
	}
}

func (client *Client) run(name string, addr string, isAutoReconnect bool, autoReconnectInterval time.Duration, tcpConnWriteChanLen int) {
	defer util.Recover()
	defer client.connsWG.Done()

reconnect:
	var conn net.Conn
	var err error

	//一直连接服务器，直到连上为止
	for {
		if client.isClosed {
			logger.WithField("name", name).Debug("tcp client conn exit dail")
			return
		}

		conn, err = net.Dial("tcp", addr)
		if err == nil {
			break
		}

		logger.WithFields(logrus.Fields{
			"name":  name,
			"addr":  addr,
			"error": err,
		}).Warning("connect error")
		time.Sleep(autoReconnectInterval)
	}

	client.connsMutex.Lock()
	if client.isClosed {
		client.connsMutex.Unlock()
		conn.Close()

		logger.WithField("name", name).Debug("tcpclient conn closed")
		return
	}

	client.index = client.index + 1
	tcpConn := newTCPConn(fmt.Sprintf("%s-%d", name, client.index), conn, false, tcpConnWriteChanLen, false, client.MsgPackager, client.MsgProcessor)
	client.conns[conn] = tcpConn
	//client.connsCnt.Add(1)
	client.connsMutex.Unlock()

	go tcpConn.WriteLoop()

	tcpConn.ReadLoop()

	// clean up
	tcpConn.Close()

	client.connsMutex.Lock()
	delete(client.conns, conn)
	//client.connsCnt.Add(-1)
	if client.isClosed {
		client.connsMutex.Unlock()

		logger.WithField("name", name).Debug("tcpclient conn exit loop")
		return
	}
	client.connsMutex.Unlock()

	if isAutoReconnect {
		time.Sleep(autoReconnectInterval)
		goto reconnect
	}
}
