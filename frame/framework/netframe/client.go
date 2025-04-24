package netframe

import (
	"github/beijian128/micius/frame/framework/worker"
	"reflect"
	"time"

	"github/beijian128/micius/frame/network/connection"
	"github/beijian128/micius/frame/network/msgprocessor"
	"github/beijian128/micius/frame/network/tcp"
)

// ClientConnectHandler ...
type ClientConnectHandler func(conn connection.Connection, ServerID uint32, uint32 uint32, ServerStartTime int64)

// Client ...
type Client struct {
	baseClient *tcp.Client

	appConfig    *AppConfig
	clientConfig *ClientConfig

	handlerIO worker.IWorker

	heartCheckTimer     *time.Timer
	heartIntervalTicker *time.Ticker

	msgHandlers         msgprocessor.GetMsgHandler
	msgHeartBeatHandler msgprocessor.MsgHandler
	msgRegisterHandler  msgprocessor.MsgHandler
}

// NewClient 创建连接客户端
func NewClient(appConfig *AppConfig, clientConfig *ClientConfig, io worker.IWorker,
	connectHandler ClientConnectHandler, closeHandler msgprocessor.CloseHandler,
	bytesHandler msgprocessor.BytesHandler, msgHandlers msgprocessor.GetMsgHandler) *Client {
	c := new(Client)

	c.appConfig = appConfig
	c.clientConfig = clientConfig
	c.handlerIO = io

	fconnect := func(conn connection.Connection) {
		//心跳时间设置
		c.heartCheckTimer = time.AfterFunc(heartBeatWaitTime, func() {
			conn.Close()
		})

		// 每隔多久发一次心跳
		c.heartIntervalTicker = time.NewTicker(heartBeatInterval)
		go func() {
			for range c.heartIntervalTicker.C {
				t := time.Now().Unix()
				c.SendMsg(conn, appConfig.ServerID, 0, 0, &Server_ReqHeartBeat{
					TimeStamp: t,
				})
			}
		}()

		// 发送请求信息
		req := &Server_ReqRegister{
			ClientType:      c.appConfig.ServerType,
			ClientId:        c.appConfig.ServerID,
			ClientStartTime: c.appConfig.StartTime,
		}
		c.SendMsg(conn, appConfig.ServerID, 0, 0, req)
	}

	fclose := func(conn connection.Connection) {
		if closeHandler != nil {
			closeHandler(conn)
		}

		if c.heartCheckTimer != nil {
			c.heartCheckTimer.Stop()
		}

		if c.heartIntervalTicker != nil {
			c.heartIntervalTicker.Stop()
		}
	}

	commonProcessor := msgprocessor.NewMetaProcessor((*Server_Extend)(nil), io)
	commonProcessor.ConnectHandler = fconnect
	commonProcessor.CloseHandler = fclose
	commonProcessor.MsgHandlers = c
	commonProcessor.BytesHandler = bytesHandler

	c.msgHeartBeatHandler = func(conn connection.Connection, ext any, _ uint32, _ []byte, msg any) {
		c.heartCheckTimer.Reset(heartBeatWaitTime)
	}
	c.msgRegisterHandler = func(conn connection.Connection, ext any, _ uint32, _ []byte, msg any) {
		resp := msg.(*Server_RespRegister)
		if connectHandler != nil {
			connectHandler(conn, resp.ServerId, resp.ServerType, resp.ServerStartTime)
		}
	}

	c.msgHandlers = msgHandlers

	c.baseClient = tcp.NewTCPClient(c.clientConfig.Name, c.clientConfig.ConnectAddr, 1, true, autoReconnetInterval, tcpCommonWriteChanLen, commonPackager, commonProcessor)
	if c.baseClient == nil {
		return nil
	}

	return c
}

// Close ...
func (c *Client) Close() {
	c.baseClient.ForEach(func(conn connection.Connection) {
		rpt := &Server_ReportUnRegister{ServerStartTime: c.appConfig.StartTime}

		c.SendMsg(conn, c.appConfig.ServerID, 0, 0, rpt)
	})
	c.baseClient.Close()
}

// SendMsg ...
func (c *Client) SendMsg(conn connection.Connection, serverID uint32, sessionId uint64, userId uint64, msg any) {
	if conn != nil {
		conn.WriteMsg(&Server_Extend{ServerId: serverID, SessionId: sessionId, UserId: userId}, msg)
	}
}

var (
	msgTypeRespHeartBeat = reflect.TypeOf((*Server_RespHeartBeat)(nil))
	msgTypeRespRegister  = reflect.TypeOf((*Server_RespRegister)(nil))
)

// GetMsgHandler ...
func (c *Client) GetMsgHandler(typ reflect.Type) (msgprocessor.MsgHandler, bool) {
	switch typ {
	case msgTypeRespHeartBeat:
		return c.msgHeartBeatHandler, true
	case msgTypeRespRegister:
		return c.msgRegisterHandler, true
	default:
		return c.msgHandlers.GetMsgHandler(typ)
	}
}
