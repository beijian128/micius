package gate

import (
	"github.com/sirupsen/logrus"
	"github/beijian128/micius/config"
	"github/beijian128/micius/frame/appframe"
	appframeslb "github/beijian128/micius/frame/appframe/slb"
	"github/beijian128/micius/frame/framework/netcluster"
	"github/beijian128/micius/services"
	_ "net/http/pprof"
)

var SessionMgrInstance *sessionManager

var AppInstance *appframe.GateApplication
var AppCfg *config.AppConfig

// InitGateSvr 初始化 gatesvr.
func InitGateSvr(app *appframe.GateApplication, cfgFile string) error {

	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return err
	}
	AppCfg = cfg
	AppInstance = app

	app.RegisterService(services.ServiceType_Lobby, appframeslb.WithLoadBalanceSingleton(app, services.ServiceType_Lobby))

	SessionMgrInstance = initSessionManager(app)

	initGateMsgRoute(app)
	initGateMsgHandler(app)

	app.OnExitHandler(Close)
	return nil
}

func Close() {
	SessionMgrInstance.close()
}

func onLobbyServerDisconnect(svrID uint32, event netcluster.SvrEvent) {
	switch event {
	case netcluster.SvrEventQuit, netcluster.SvrEventDisconnect:
		logrus.WithFields(logrus.Fields{
			"svrID": svrID,
			"event": event,
		}).Error("lobby server disconnect")

		SessionMgrInstance.execByEverySession(func(s *session) {
			s.Close()
		})
	}
}
