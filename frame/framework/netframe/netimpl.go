package netframe

import (
	"fmt"
	"github/beijian128/micius/frame/framework/worker"
	"net"
	"reflect"
	"sync"

	"github/beijian128/micius/frame/network/msgprocessor"

	"github.com/sirupsen/logrus"

	"github/beijian128/micius/frame/network/connection"
)

// implMetaNet ...
// 通用网络接口的实现
type implMetaNet struct {
	// 配置
	Config *AppConfig
	// 事件
	HandlerIO worker.IWorker
	// 消息管理
	ConnectHandler OnNetConnect
	CloseHandler   OnNetClose
	BytesHandler   OnNetBytes
	//
	WorkerExitHandler OnWorkerExit

	// 服务和客户端
	NetMutex sync.Mutex
	Clients  map[string]*Client
	Servers  map[string]*Server

	// 连接管理, map find是goroutine安全的， 所以允许在别的goroutine发送消息不加锁
	ID2ConnItem   map[uint32]*ConnItem
	Conn2ConnItem map[connection.Connection]*ConnItem

	*msgprocessor.MsgHandlers
}

// ConnItem ...
type ConnItem struct {
	ID              uint32
	ServerType      uint32
	ServerStartTime int64

	_conn connection.Connection
}

// 是否时服务的连接
func (s *ConnItem) isServiceConn() bool {
	return s.ServerType != 0
}

func (s *ConnItem) getConn() (connection.Connection, bool) {
	if s._conn == nil {
		return nil, false
	}
	return s._conn, true
}

// NewMetaNet ...
func NewMetaNet() MetaNet {
	Net := new(implMetaNet)

	Net.MsgHandlers = msgprocessor.NewMsgHandlers()

	Net.Clients = make(map[string]*Client)
	Net.Servers = make(map[string]*Server)
	Net.ID2ConnItem = make(map[uint32]*ConnItem)
	Net.Conn2ConnItem = make(map[connection.Connection]*ConnItem)

	return Net
}

func (net *implMetaNet) Init(config *AppConfig, io worker.IWorker) {
	if config == nil {
		panic("MetaNet init Err: config==nil")
	}
	if !IsServerID(config.ServerID) {
		panic("MetaNet init Err: config.ServerID is illegal")
	}
	if io == nil {
		panic("MetaNet init Err: io==nil")
	}

	net.Config = config
	net.HandlerIO = io

	initMessage()
}

func (net *implMetaNet) ListenConnect(fconnect OnNetConnect) {
	net.ConnectHandler = fconnect
}

func (net *implMetaNet) ListenClose(fclose OnNetClose) {
	net.CloseHandler = fclose
}

func (net *implMetaNet) ListenBytes(fbytes OnNetBytes) {
	net.BytesHandler = fbytes
}

func (net *implMetaNet) ListenMessage(msg any, fmessage OnNetMessage) {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		panic("MetaNet ListenMessage, message pointer required")
	}
	if fmessage == nil {
		panic("ListenMessage handler is nil")
	}

	handler := func(conn connection.Connection, ext any, msgId uint32, msgData []byte, msg any) {
		connItem, ok := net.findConnItem(conn)
		if !ok {
			logger.WithFields(logrus.Fields{
				"conn":    conn,
				"msgtype": reflect.TypeOf(msg),
			}).Warning("MetaNet onMessage no connitem")
			return
		}

		var extend Server_Extend
		if ext != nil {
			if svrExt, ok := ext.(*Server_Extend); ok {
				extend = *(svrExt)
			} else {
				if extData, ok := ext.([]byte); ok {
					extend = Server_Extend{ClientData: extData}
				}
			}
		}

		fmessage(connItem.ID, connItem.ServerType, msgId, msgData, msg, extend)
	}

	net.MsgHandlers.AddHandler(msg, handler)
}

func (net *implMetaNet) ListenWorkerExit(fexit OnWorkerExit) {
	net.WorkerExitHandler = fexit
}

func (net *implMetaNet) Connect(config *ClientConfig) {
	net.NetMutex.Lock()
	defer net.NetMutex.Unlock()

	if _, ok := net.Clients[config.ConnectAddr]; ok {
		logger.WithField("addr", config.ConnectAddr).Error("MetaNet Connect Addr has used")
		return
	}

	client := NewClient(net.Config, config, net.HandlerIO, net.onConnect, net.onClose, net.onBytes, net)
	if client == nil {
		panic(fmt.Sprintf("MetaNet Connect failed, config:%v", config))
	}

	net.Clients[config.ConnectAddr] = client

	logger.WithField("addr", config.ConnectAddr).Info("MetaNet Connect start")
}

func (net *implMetaNet) Listen(config *ServerConfig, isGate bool) {
	net.NetMutex.Lock()
	defer net.NetMutex.Unlock()

	if _, ok := net.Servers[config.ListenAddr]; ok {
		logger.WithField("addr", config.ListenAddr).Error("MetaNet Server Addr has uesd")
		return
	}

	var server *Server

	if isGate {
		server = NewGateServer(net.Config, config, net.HandlerIO, net.onConnect, net.onClose, net.onBytes, net)
	} else {
		server = NewCommonServer(net.Config, config, net.HandlerIO, net.onConnect, net.onClose, net.onBytes, net, net.onWorkerExit)
	}

	if server == nil {
		panic(fmt.Sprintf("MetaNet Server failed, config:%v", config))
	}

	net.Servers[config.ListenAddr] = server

	logger.WithField("addr", config.ListenAddr).Info("MetaNet Listen start")
}

func (net *implMetaNet) Close(ID uint32) {
	net.NetMutex.Lock()
	defer net.NetMutex.Unlock()

	if connItem, ok := net.ID2ConnItem[ID]; ok {
		if c, ok := connItem.getConn(); ok {
			c.Close()
		}
	}
}

func (net *implMetaNet) Fini() {
	net.NetMutex.Lock()
	defer net.NetMutex.Unlock()

	for _, c := range net.Clients {
		c.Close()
	}

	for _, s := range net.Servers {
		s.Close()
	}
}

func (net *implMetaNet) Post(f func()) {
	net.HandlerIO.Post(f)
}

func (net *implMetaNet) LocalAddr(ID uint32) net.Addr {
	net.NetMutex.Lock()
	defer net.NetMutex.Unlock()

	if connItem, ok := net.ID2ConnItem[ID]; ok {
		if c, ok := connItem.getConn(); ok {
			return c.LocalAddr()
		}
	}
	return nil
}

func (net *implMetaNet) RemoteAddr(ID uint32) net.Addr {
	net.NetMutex.Lock()
	defer net.NetMutex.Unlock()

	if connItem, ok := net.ID2ConnItem[ID]; ok {
		if c, ok := connItem.getConn(); ok {
			return c.RemoteAddr()
		}
	}
	return nil
}

func (net *implMetaNet) SendGateMsg(ID uint32, msg any) error {
	return net.SendMsg(ID, msg, nil)
}

func (net *implMetaNet) SendGateBytes(ID uint32, msgid uint32, bytes []byte) error {
	return net.SendBytes(ID, msgid, bytes, nil)
}

func (net *implMetaNet) SendMsg(ID uint32, msg any, extend *Server_Extend) error {
	net.NetMutex.Lock()
	defer net.NetMutex.Unlock()

	if connItem, ok := net.ID2ConnItem[ID]; ok {
		if conn, ok := connItem.getConn(); ok {
			if IsServerID(ID) {
				return conn.WriteMsg(extend, msg)
			}
			return conn.WriteMsg(extend, msg)
		}
	}

	return fmt.Errorf("MetaNet SendMsg, no serverid:%d, err id or has disconnected", ID)
}

func (net *implMetaNet) SendBytes(ID uint32, msgid uint32, bytes []byte, extend *Server_Extend) error {
	net.NetMutex.Lock()
	defer net.NetMutex.Unlock()

	if connItem, ok := net.ID2ConnItem[ID]; ok {
		if conn, ok := connItem.getConn(); ok {
			if IsServerID(ID) {
				return conn.WriteBytes(extend, msgid, bytes)
			}
			return conn.WriteBytes(extend.ClientData, msgid, bytes)
		}
	}

	return fmt.Errorf("MetaNet SendBytes, no serverid:%d, err id or has disconnected", ID)
}

func (net *implMetaNet) onConnect(conn connection.Connection, ServerID uint32, ServerType uint32, ServerStartTime int64) {
	logger.WithFields(logrus.Fields{
		"local":   conn.Name(),
		"svrid":   ServerID,
		"svrtype": ServerType,
	}).Trace("MetaNet OnConnect")

	if ServerID == net.Config.ServerID || ServerType == net.Config.ServerType {
		conn.Close()
		logger.WithFields(logrus.Fields{
			"svrid":   ServerID,
			"svrtype": ServerType,
			"conn":    conn,
		}).Warning("MetaNet OnConnect client can not connect it's server!!!")
		return
	}

	var reboot bool

	net.NetMutex.Lock()

	item, exist := net.ID2ConnItem[ServerID]
	if exist {
		if _, ok := item.getConn(); ok {
			// 旧的连接还在，不应该让新的上来。
			conn.Close()
			logger.WithFields(logrus.Fields{
				"svrid":     ServerID,
				"svrtype":   ServerType,
				"starttime": ServerStartTime,
			}).Warning("MetaNet OnConnect has connected")
			net.NetMutex.Unlock()
			return
		}
		if item.ServerType != ServerType {
			conn.Close()
			logger.WithFields(logrus.Fields{
				"svrid":     ServerID,
				"svrtype":   ServerType,
				"starttime": ServerStartTime,
			}).Warning("MetaNet OnConnect ServerType diff")
			net.NetMutex.Unlock()
			return
		}
		// 服务重启.
		if item.ServerStartTime != ServerStartTime {
			reboot = true
		}
		item._conn = conn
		item.ServerStartTime = ServerStartTime
	} else {
		item = &ConnItem{
			_conn:           conn,
			ID:              ServerID,
			ServerType:      ServerType,
			ServerStartTime: ServerStartTime,
		}
		net.ID2ConnItem[ServerID] = item
	}
	net.Conn2ConnItem[conn] = item

	net.NetMutex.Unlock()

	// 通知上层:
	// 	1.重连（保证之前已通知过断开连接)
	//	2.新服务上线 (如果是重启，保证之前已通知过旧服务下线)

	if reboot {
		if net.WorkerExitHandler != nil {
			net.WorkerExitHandler(item.ID, item.ServerType)
		}
	}

	if net.ConnectHandler != nil {
		net.ConnectHandler(ServerID, ServerType)
	}
}

func (net *implMetaNet) onClose(conn connection.Connection) {
	net.NetMutex.Lock()

	item, ok := net.findConnItemLocked(conn)
	if !ok {
		net.NetMutex.Unlock()
		return
	}

	logger.WithFields(logrus.Fields{
		"local":   conn.Name(),
		"svrid":   item.ID,
		"svrtype": item.ServerType,
	}).Debug("MetaNet onClose")

	net.delConnLocked(conn)

	net.NetMutex.Unlock()

	if net.CloseHandler != nil {
		net.CloseHandler(item.ID, item.ServerType)
	}
}

func (net *implMetaNet) onBytes(conn connection.Connection, ext any, msgid uint32, bytes []byte) {
	if net.BytesHandler == nil {
		return
	}

	connItem, ok := net.findConnItem(conn)
	if !ok {
		logger.WithFields(logrus.Fields{
			"conn":  conn,
			"msgid": msgid,
		}).Warning("MetaNet onBytes no connitem", conn, msgid)
		return
	}

	if ext != nil {

		switch ext.(type) {
		case *Server_Extend:
			exts := ext.(*Server_Extend)
			net.BytesHandler(connItem.ID, connItem.ServerType, msgid, bytes, *exts)
		case []byte:
			net.BytesHandler(connItem.ID, connItem.ServerType, msgid, bytes, Server_Extend{ClientData: ext.([]byte)})
			logger.WithFields(logrus.Fields{
				"conn":  conn,
				"msgid": msgid,
			}).Warning("MetaNet onBytes  ext type is []byte", conn, msgid)
		default:
			logger.WithFields(logrus.Fields{
				"conn":  conn,
				"msgid": msgid,
			}).Warning("MetaNet onBytes unknown ext type", conn, msgid)
		}

	} else {
		net.BytesHandler(connItem.ID, connItem.ServerType, msgid, bytes, Server_Extend{})
	}
}

func (net *implMetaNet) onWorkerExit(conn connection.Connection, ServerStartTime int64) {
	net.NetMutex.Lock()

	item, ok := net.findConnItemLocked(conn)
	if !ok {
		net.NetMutex.Unlock()
		return
	}

	net.delConnItemLocked(item.ID)

	net.NetMutex.Unlock()

	if net.WorkerExitHandler != nil {
		net.WorkerExitHandler(item.ID, item.ServerType)
	}
}

func (net *implMetaNet) findConnItem(conn connection.Connection) (*ConnItem, bool) {
	net.NetMutex.Lock()
	defer net.NetMutex.Unlock()
	return net.findConnItemLocked(conn)
}

func (net *implMetaNet) findConnItemLocked(conn connection.Connection) (*ConnItem, bool) {
	item, ok := net.Conn2ConnItem[conn]
	return item, ok
}

func (net *implMetaNet) delConnItemLocked(svrid uint32) {
	if item, ok := net.ID2ConnItem[svrid]; ok {
		delete(net.ID2ConnItem, svrid)
		delete(net.Conn2ConnItem, item._conn)
	}
}

func (net *implMetaNet) delConnLocked(conn connection.Connection) {
	if item, ok := net.Conn2ConnItem[conn]; ok {
		if item.isServiceConn() {
			item._conn = nil
		} else {
			delete(net.ID2ConnItem, item.ID)
		}
		delete(net.Conn2ConnItem, conn)
	}
}
