package main

import (
	"flag"
	"github.com/sirupsen/logrus"
	"github/beijian128/micius/frame/appframe"
	"github/beijian128/micius/frame/appframe/master"
	"github/beijian128/micius/frame/framework/netcluster"
	"github/beijian128/micius/frame/log"
	"github/beijian128/micius/frame/util"
	"github/beijian128/micius/services/gate"
	"github/beijian128/micius/services/lobby"
	"net/http"
	"sync"
)

var (
	help          = flag.Bool("h", false, "help")
	netconfigFile = flag.String("netconfig", "netconfig.json", "netconfig file")
	appconfigFile = flag.String("config", "app.yaml", "app config")
	noMaster      = flag.Bool("noMaster", false, "ignore Master server")
	noGate        = flag.Bool("noGate", false, "ignore gate server")
)

var (
	masterName = "master"
	gateName   = "gate"

	lobbyName = "lobby"
)

func init() {
	master.DisableMasterInitGlobalLogrus = true
	appframe.DisableApplicationInitGlobalLogrus = true
}

func main() {

	flag.Parse()
	if *help {
		flag.Usage()
		return
	}

	log.InitLogrus(&log.Config{
		Name:  "chat",
		Level: 5,
		Outputs: map[string]map[string]interface{}{
			"file": map[string]interface{}{
				"path":   "./logs",
				"rotate": true,
			},
		},
	})

	netconfig, err := netcluster.ParseClusterConfigFile(*netconfigFile)
	if err != nil {
		logrus.WithError(err).Panic("netconfigFile.load", err)
		return
	}

	findOneNode := func(base string) string {
		list := []string{base, base + "1"}
		for _, target := range list {
			for nodeName, _ := range netconfig.Masters {
				if nodeName == target {
					return target
				}
			}
			for nodeName, _ := range netconfig.Slaves {
				if nodeName == target {
					return target
				}
			}
		}
		return base
	}

	wg := sync.WaitGroup{}

	// master
	if !*noMaster {
		wg.Add(1)
		util.SafeGo(func() {
			defer wg.Done()
			m, err := master.New(*netconfigFile, findOneNode(masterName))
			if err != nil {
				logrus.WithField("name", masterName).WithError(err).Panic("New master fail")
			}
			m.Run()
		})
	}

	if !*noGate {
		// gate
		wg.Add(1)
		util.SafeGo(func() {
			defer wg.Done()
			app, err := appframe.NewGateApplication(*netconfigFile, findOneNode(gateName))
			if err != nil {
				logrus.WithField("name", gateName).WithError(err).Panic("New gate app fail")
			}
			err = gate.InitGateSvr(app, *appconfigFile)
			if err != nil {
				logrus.WithField("name", gateName).WithError(err).Panic("Init gatesvr fail")
			}
			app.Run()
		})
	}

	//lobby
	wg.Add(1)
	util.SafeGo(func() {
		defer wg.Done()
		app, err := appframe.NewApplication(*netconfigFile, findOneNode(lobbyName))
		if err != nil {
			logrus.WithField("name", lobbyName).WithError(err).Panic("New lobby app fail")
		}
		err = lobby.InitLobbySvr(app, *appconfigFile)
		if err != nil {
			logrus.WithField("name", lobbyName).WithError(err).Panic("Init lobbysvr fail")
		}
		app.Run()
	})

	http.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("web"))))
	http.ListenAndServe(":8080", nil)

	wg.Wait()
}
