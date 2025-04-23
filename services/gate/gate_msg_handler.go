package gate

import (
	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
	"github/beijian128/micius/frame/appframe"
	"github/beijian128/micius/frame/framework/netframe"
	"github/beijian128/micius/proto/cmsg"
	"github/beijian128/micius/services"
)

func initGateMsgHandler(app *appframe.GateApplication) {
 
}

func initGateMsgRoute(app *appframe.GateApplication) {
	logrus.Info("------------------initGateMsgRoute ---------------------")

	//lobby
	lobby := app.GetService(services.ServiceType_Lobby)
	RouteSessionRowMsg(app, (*cmsg.CReqLogin)(nil), lobby)
	RouteSessionRowMsg(app, (*cmsg.CReqSendChatMessage)(nil), lobby)

}

func GetAnyRawMessageRouter(f func(s *session, msgid uint32, data []byte, extParam int64)) appframe.GateSessionRawMsgRouter {
	return func(sid uint64, msgid uint32, data []byte, extParam int64) {
		s, ok := SessionMgrInstance.getSession(sid)
		if ok {
			f(s, msgid, data, extParam)
		}
	}
}

func RouteSessionRowMsg(app *appframe.GateApplication, msg proto.Message, service appframe.Service) {
	app.RouteSessionRawMsg(msg, GetAnyRawMessageRouter(func(s *session, mssgid uint32, data []byte, extParam int64) {
		service.ForwardRawMsgFromSession(mssgid, data, netframe.Server_Extend{
			SessionId: s.ID(),
			UserId:    s.userid,
			ExtParam:  extParam,
		})
	}))
}
