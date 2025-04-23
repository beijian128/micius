package netcluster

import (
	"github/beijian128/micius/frame/framework/netframe"
	"github/beijian128/micius/frame/framework/worker"
	"net"
	"time"

	"github.com/sirupsen/logrus"

	"strings"
)

var logger = logrus.WithField("where", "netcluster/master")

type Master struct {
	NetIO         worker.IWorker
	ClusterConfig *ClusterConf
	MasterConfig  *MasterConf

	PrintLoadLevelStatus bool

	net netframe.MetaNet
	// all id->server
	id2servers map[uint32]*ServerItem
	// all type->server
	type2servers map[uint32]*ServerGroup
	//负载
	loadLevel uint32
	//router info
	routerA2B          map[uint32]map[uint32]*routerItem
	printMsgStatTicker *time.Ticker
	// slave热更新
	cfgCheckMap map[uint32]*cfgCheckResult
}

type cfgCheckResult struct {
	timestamp int64
	cfgMd5    string
	result    map[uint32]bool
}

// ServerGroup 同type的 所有服务
type ServerGroup struct {
	serverType uint32
	// 被哪些服务关注
	subId2servers map[uint32]*ServerItem
	// 有哪些服务
	id2servers map[uint32]*ServerItem
}

// ServerItem 服务节点
type ServerItem struct {
	config *SlaveConf
	cfgMd5 string

	isConnected bool
	isWorking   bool

	disableNewConsistentFlow bool
}

type routerItem struct {
	msgTotal uint64 // 消息总数
	msgCnt   uint32 // 仅限x分钟内的消息总数 （暂定x=5）
}

// NewMaster ...
func NewMaster(config *ClusterConf, key string, io worker.IWorker) *Master {
	mcfg, bhave := config.Masters[key]
	if !bhave {
		logger.WithField("server", key).Panic("NewMaster can not find server")
	}

	m := &Master{}
	m.NetIO = io
	m.ClusterConfig = config
	m.MasterConfig = mcfg
	m.PrintLoadLevelStatus = mcfg.PrintLoadLevelStatus

	return m
}

// Init 初始化
func (m *Master) Init() {

	m.net = netframe.NewMetaNet()
	m.id2servers = make(map[uint32]*ServerItem)
	m.type2servers = make(map[uint32]*ServerGroup)
	m.routerA2B = make(map[uint32]map[uint32]*routerItem)
	m.cfgCheckMap = make(map[uint32]*cfgCheckResult)

	for _, sconf := range m.ClusterConfig.Slaves {
		sitem := &ServerItem{config: sconf, isConnected: false, isWorking: false}
		var group *ServerGroup

		// set id2servers
		m.id2servers[sconf.ServerID] = sitem

		// set all type2servers
		if g, ok := m.type2servers[sitem.config.ServerType]; ok {
			group = g
		} else {
			group = &ServerGroup{}
			group.serverType = sitem.config.ServerType
			group.subId2servers = make(map[uint32]*ServerItem)
			group.id2servers = make(map[uint32]*ServerItem)

			m.type2servers[sitem.config.ServerType] = group
		}
		group.id2servers[sitem.config.ServerID] = sitem
	}

	// set subId2servers
	for id, sitem := range m.id2servers {
		for _, subtype := range sitem.config.SubscribedTypes {
			if group, ok := m.type2servers[subtype]; ok {
				group.subId2servers[id] = sitem
			}
		}
	}

	m.net.Init(&netframe.AppConfig{
		ServerID:   m.MasterConfig.ServerID,
		ServerType: m.MasterConfig.ServerType,
		StartTime:  time.Now().Unix(),
	}, m.NetIO)

	m.net.ListenConnect(m.OnConnect)
	m.net.ListenClose(m.OnClose)
	m.net.ListenBytes(m.OnBytes)
	m.net.ListenWorkerExit(m.OnWorkerExit)
	m.net.ListenMessage((*Slave_ReportLoadLevel)(nil), m.onSlaveReportLoadLv)
	m.net.ListenMessage((*Slave_ReqVerifyConfigFile)(nil), m.onSlaveReqVerifyConfig)
	m.net.ListenMessage((*Slave_RepVerifyConfigFile)(nil), m.onSlaveRepVerifyConfigFile)
	m.net.ListenMessage((*Slave_UptConfigMd5)(nil), m.onSlaveUptConfigMd5)
	m.net.ListenMessage((*SM_ReqPrepareCloseMyself)(nil), m.onSlavePrepareClose)
}

// Run ...
func (m *Master) Run() {
	sc := &netframe.ServerConfig{
		Name:       "MasterServer",
		ListenAddr: m.MasterConfig.ListenAddr,
		MaxConnCnt: m.MasterConfig.MaxConnCnt,
	}

	m.net.Listen(sc, false)

	m.printRMsgStat()
}

// Fini ...
func (m *Master) Fini() {
	m.net.Fini()

	if m.printMsgStatTicker != nil {
		m.printMsgStatTicker.Stop()
	}
}

// Post ...
func (m *Master) Post(f func()) {
	m.net.Post(f)
}

// LocalAddr ...
func (m *Master) LocalAddr(ID uint32) net.Addr {
	return m.net.LocalAddr(ID)
}

// RemoteAddr ...
func (m *Master) RemoteAddr(ID uint32) net.Addr {
	return m.net.RemoteAddr(ID)
}

// SendMsg ...
func (m *Master) SendMsg(ID uint32, msg interface{}, extend *netframe.Server_Extend) error {
	return m.net.SendMsg(ID, msg, extend)
}

// SendBytes ...
func (m *Master) SendBytes(ID uint32, msgid uint32, bytes []byte, extend *netframe.Server_Extend) error {
	return m.net.SendBytes(ID, msgid, bytes, extend)
}

// Close ...
func (m *Master) Close(ID uint32) {
	m.net.Close(ID)
}

// OnConnect ...
func (m *Master) OnConnect(ID uint32, serverType uint32) {
	logger.WithFields(logrus.Fields{
		"svrid":   ID,
		"svrtype": serverType,
	}).Info("[Master] Slave Connected, wait for config check")
}

// OnClose ...
func (m *Master) OnClose(ID uint32, serverType uint32) {
	logger.WithFields(logrus.Fields{
		"svrid":   ID,
		"svrtype": serverType,
	}).Info("[Master] Slave Closed.")

	if server, ok := m.id2servers[ID]; ok {
		server.isConnected = false

		// 通知关注我的服务
		m.publish2MySubscribers(ID, serverType)

		//load level
		m.routerRemoveA2All(ID)
		m.routerRemoveAll2B(ID)
	}

	// 连接关闭则清理此数据
	delete(m.cfgCheckMap, ID)
}

func (m *Master) doNewSlaveConnect(ID uint32, serverType uint32, serverCfgMd5 string) {
	// 处理新slave时,其配置才真正生效
	if !m.addNewSalveCfg(ID) {
		m.Close(ID)
		logger.WithField("svrid", ID).Warning("OnConnect Master server is illegality")
		return
	}

	if server, ok := m.id2servers[ID]; ok {
		if server.config.ServerType != serverType {
			m.Close(ID)
			logger.WithFields(logrus.Fields{
				"svrid":      ID,
				"svrtype":    server.config.ServerType,
				"newSvrType": serverType,
			}).Warning("OnConnect Master server conflict type")
		} else {
			server.cfgMd5 = serverCfgMd5
			server.isConnected = true
			server.isWorking = true

			// 告诉slave, master的 load level
			m.publishLoadLevel(ID)

			// 我关注的服务状态通知给我
			m.publishSubscribed2Me(server.config.SubscribedTypes, ID)

			// 通知关注我的服务
			m.publish2MySubscribers(ID, serverType)

			logger.WithFields(logrus.Fields{
				"svrid":   ID,
				"svrtype": serverType,
			}).Info("[Master] Slave 连接成功.")
		}
	} else {
		m.Close(ID)
		logger.WithField("svrid", ID).Warning("OnConnect Master server is illegality")
	}
}

func (m *Master) canLoadNewConfig(newConfig *ClusterConf) bool {
	if newConfig == nil {
		return false
	}

	if !newConfig.IsSameMasterCfg(m.MasterConfig) {
		return false
	}

	for _, sitem := range m.id2servers {
		if !newConfig.IsSameSlaveCfg(sitem.config) {
			return false
		}
	}

	return true
}

// isSameCfgAllCluster 已经激活的服务其配置是否都和master相同
func (m *Master) isSameCfgAllCluster() bool {
	for _, sitem := range m.id2servers {
		if sitem.isConnected || sitem.isWorking {
			if strings.Compare(sitem.cfgMd5, m.ClusterConfig.FileMd5) != 0 {
				return false
			}
		}
	}
	return true
}

// 检查配置并执行连接
func (m *Master) onSlaveReqVerifyConfig(ID uint32, serverType uint32, _ uint32, _ []byte, msg interface{}, extend netframe.Server_Extend) {
	req := msg.(*Slave_ReqVerifyConfigFile)
	logger := logger.WithFields(logrus.Fields{
		"svrid":     ID,
		"master id": m.MasterConfig.ServerID,
		"where":     "onSlaveReqVerifyConfig",
	})

	// master检查配置md5
	logger.Trace("received message:", req)

	if strings.Compare(req.FileMd5, m.ClusterConfig.FileMd5) != 0 {

		logger.Tracef("新连接的slave使用了新版netconfig（%s）", req.FileMd5)

		//新配置
		if newConfig, err := m.ClusterConfig.LoadNewCfgFile(); err == nil {
			logger.Debugf("master成功加载新版 netconfig（%s）", newConfig.FileMd5)
			if strings.Compare(req.FileMd5, newConfig.FileMd5) == 0 && m.canLoadNewConfig(newConfig) {
				m.ClusterConfig = newConfig

				logger.Debugf("master使用新版 netconfig（%s），并通知其他激活的slave也使用新版netconfig", newConfig.FileMd5)

				goto CheckAll
			}
		}

		m.Close(ID)
		logger.Error("OnConnect Master slave与master集群配置文件不一致")
		return
	}

CheckAll:
	if !m.isSameCfgAllCluster() {
		// 通知其他激活的slave加载新配置
		if _, ok := m.cfgCheckMap[ID]; ok {
			m.Close(ID)
			logger.Error("OnConnect Master slave重复启动，请10秒后在试!")
			return
		}
		ts := time.Now().UnixNano()
		result := &cfgCheckResult{timestamp: ts, cfgMd5: req.FileMd5, result: make(map[uint32]bool)}
		m.cfgCheckMap[ID] = result

		// 通知其他slave校验配置
		for _, sitem := range m.id2servers {
			if sitem.isConnected || sitem.isWorking {

				logger.Tracef("master通知 svr(id=%d, %s) 加载新netconfig(%s)", sitem.config.ServerID, sitem.cfgMd5, req.FileMd5)

				m.SendMsg(sitem.config.ServerID, &Master_ReqVerifyConfigFile{FileMd5: req.FileMd5, Time: ts, ReqServerId: ID, ReqServerType: serverType}, &netframe.Server_Extend{
					ServerId: m.MasterConfig.ServerID})
			}
		}

		// 超时处理
		m.NetIO.AfterPost(time.Second*10, func() {
			delete(m.cfgCheckMap, ID)

			if sitem, ok := m.id2servers[ID]; ok {
				if sitem.isConnected || sitem.isWorking {
					return
				}
			}
			m.Close(ID)
			logger.WithField("svrid", ID).Error("OnConnect Master slave配置校验超时")
		})
		return
	}

	// 执行连接
	m.doNewSlaveConnect(ID, serverType, req.FileMd5)
}

func (m *Master) onSlaveRepVerifyConfigFile(ID uint32, serverType uint32, _ uint32, _ []byte, msg interface{}, extend netframe.Server_Extend) {
	req := msg.(*Slave_RepVerifyConfigFile)

	if re, ok := m.cfgCheckMap[req.ReqServerId]; ok {
		if re.timestamp == req.Time {
			re.result[ID] = req.IsSucc

			// 是否所有slave(此前已注册的服务)都载入新配置成功
			isCfgAllOk := true
			for _, sitem := range m.id2servers {

				if !(sitem.isConnected || sitem.isWorking) {
					// 这里continue的原因
					// 1.还未注册的服务（即将注册的新服务）的配置校验在onSlaveReqVerifyConfig 里已经完成，没必要再做一次。
					// 2.如果不跳过，会导致新服务的注册超时
					continue
				}

				if isCfgOk, ok := re.result[sitem.config.ServerID]; ok {
					if !isCfgOk {
						isCfgAllOk = false
						break
					}
				} else {
					isCfgAllOk = false
					break
				}
			}

			if isCfgAllOk {
				// master配置没变
				if strings.Compare(m.ClusterConfig.FileMd5, re.cfgMd5) == 0 {
					m.doNewSlaveConnect(ID, serverType, re.cfgMd5)
				}
				return
			}
		}
	}
}

func (m *Master) onSlaveUptConfigMd5(ID uint32, serverType uint32, _ uint32, _ []byte, msg interface{}, extend netframe.Server_Extend) {
	req := msg.(*Slave_UptConfigMd5)

	if sitem, ok := m.id2servers[ID]; ok {
		if sitem.config.ServerType == serverType {
			sitem.cfgMd5 = req.FileMd5
		}
	}
}

// OnBytes Master 收到A服务的消息后转发给B服务 （出于运行效率考虑，不做关注判断，即B是否为A的关注对象）
func (m *Master) OnBytes(ID uint32, serverType uint32, msgid uint32, bytes []byte, extend netframe.Server_Extend) {
	fromServerID := ID
	toServerID := extend.ServerId

	extend.ServerId = fromServerID

	//logger.Tracef("------master router bytes to:%d, from:%d, userid:%d, msgid:%d", toServerID, fromServerID, extend.UserId, msgid)
	err := m.SendBytes(toServerID, msgid, bytes, &extend)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"msgid":     msgid,
			"fromSvrid": fromServerID,
			"toSvrid":   toServerID,
			"error":     err,
		}).Warning("master router bytes error", msgid, fromServerID, toServerID, err)
		return
	}

	// 消息转发情况统计
	if sb, ok := m.routerA2B[fromServerID]; ok {
		if item, bok := sb[toServerID]; bok {
			item.msgTotal++
			item.msgCnt++
		}
	}
}

// OnWorkerExit ...
func (m *Master) OnWorkerExit(svrid uint32, serverType uint32) {
	logger.WithFields(logrus.Fields{
		"svrid":   svrid,
		"svrtype": serverType,
	}).Info("Slave-Connect WorkerExit.")

	if sitem, ok := m.id2servers[svrid]; ok {
		sitem.isWorking = false
		m.publish2MySubscribers(sitem.config.ServerID, sitem.config.ServerType)
	}
}

func (m *Master) publishSubscribed2Me(SubscribedTypes []uint32, Subscriber uint32) {
	for _, subType := range SubscribedTypes {
		if g, ok := m.type2servers[subType]; ok {
			for _, sitem := range g.id2servers {
				if sitem.isConnected {
					msg := &Master_PublishServerStatus{
						ServerId:                 sitem.config.ServerID,
						ServerType:               sitem.config.ServerType,
						IsConnected:              sitem.isConnected,
						IsWorking:                sitem.isWorking,
						DisableNewConsistentFLow: sitem.disableNewConsistentFlow,
					}

					m.SendMsg(Subscriber, msg, &netframe.Server_Extend{ServerId: m.MasterConfig.ServerID})
				}
			}
		}
	}
}

func (m *Master) publish2MySubscribers(serverID uint32, serverType uint32) {
	if group, okg := m.type2servers[serverType]; okg {
		if sitem, oks := group.id2servers[serverID]; oks {
			msg := &Master_PublishServerStatus{
				ServerId:                 sitem.config.ServerID,
				ServerType:               sitem.config.ServerType,
				IsConnected:              sitem.isConnected,
				IsWorking:                sitem.isWorking,
				DisableNewConsistentFLow: sitem.disableNewConsistentFlow,
			}
			for _, follower := range group.subId2servers {
				if follower.isConnected {
					m.SendMsg(follower.config.ServerID, msg, &netframe.Server_Extend{ServerId: m.MasterConfig.ServerID})
				}
			}
		}
	}
}

func (m *Master) publish2Target(serverID uint32, serverType uint32, targetID uint32) {
	if group, okg := m.type2servers[serverType]; okg {
		if sitem, oks := group.id2servers[serverID]; oks {
			msg := &Master_PublishServerStatus{
				ServerId:                 sitem.config.ServerID,
				ServerType:               sitem.config.ServerType,
				IsConnected:              sitem.isConnected,
				IsWorking:                sitem.isWorking,
				DisableNewConsistentFLow: sitem.disableNewConsistentFlow,
			}

			m.SendMsg(targetID, msg, &netframe.Server_Extend{ServerId: m.MasterConfig.ServerID})
		}
	}
}

func (m *Master) routerAddA2B(aID uint32, bID uint32) {
	if sb, ok := m.routerA2B[aID]; ok {
		if _, bok := sb[bID]; bok {
			logger.WithFields(logrus.Fields{
				"a":      aID,
				"master": m.MasterConfig.ServerID,
				"b":      bID,
			}).Warningf("[routerAddA2B] had added, A:%d----%d---->B:%d", aID, m.MasterConfig.ServerID, bID)
			return
		}

		sb[bID] = &routerItem{}
	} else {
		sb := make(map[uint32]*routerItem)
		sb[bID] = &routerItem{}
		m.routerA2B[aID] = sb
	}

	tmpw := uint32(5)
	var aname, bname string
	if ainfo, aok := m.id2servers[aID]; aok {
		aname = ainfo.config.ServerName
		//log.BDebug("slaveweights: %v", m.MasterConfig.SlaveWeights[ainfo.config.ServerType])
		if aw, wok := m.MasterConfig.SlaveWeights[ainfo.config.ServerType]; wok {
			tmpw = aw
		}
	}
	if binfo, bok := m.id2servers[bID]; bok {
		bname = binfo.config.ServerName
	}

	m.loadLevel += tmpw
	m.publishLoadLevel(0)

	logger.WithFields(logrus.Fields{
		"master loadLevel": m.loadLevel,
	}).Infof("[FixM]连接 %s(%d) --> %s(%d) --> %s(%d)", aname, aID, m.MasterConfig.ServerName, m.MasterConfig.ServerID, bname, bID)
}

func (m *Master) routerRemoveA2B(aID uint32, bID uint32) {
	if sb, ok := m.routerA2B[aID]; ok {
		if _, bok := sb[bID]; bok {
			var aname, bname string
			tmpw := uint32(5)
			if ainfo, aok := m.id2servers[aID]; aok {
				aname = ainfo.config.ServerName
				if aw, wok := m.MasterConfig.SlaveWeights[ainfo.config.ServerType]; wok {
					tmpw = aw
				}
			}
			if binfo, bok := m.id2servers[bID]; bok {
				bname = binfo.config.ServerName
			}

			if m.loadLevel < tmpw {
				m.loadLevel = 0
			} else {
				m.loadLevel -= tmpw
			}
			m.publishLoadLevel(0)

			delete(sb, bID)

			logger.WithFields(logrus.Fields{
				"master loadLevel": m.loadLevel,
			}).Infof("[FixM]断开 %s(%d) --> %s(%d) --> %s(%d)",
				aname, aID, m.MasterConfig.ServerName, m.MasterConfig.ServerID, bname, bID)

			return
		}
	}
}

func (m *Master) routerRemoveAll2B(bID uint32) {
	for aID, bmap := range m.routerA2B {
		if _, ok := bmap[bID]; ok {
			m.routerRemoveA2B(aID, bID)
		}
	}
}

func (m *Master) routerRemoveA2All(aID uint32) {
	if sb, ok := m.routerA2B[aID]; ok {
		for bID := range sb {
			m.routerRemoveA2B(aID, bID)
		}
	}
}

// Master将自己的负载情况通知给目标server（特别地，当serverID==0时，代表通知所有服务）
func (m *Master) publishLoadLevel(serverID uint32) {
	msg := &Master_PublishLoadLevel{
		LoadLevel: m.loadLevel,
	}

	if serverID == 0 {
		for id, sitem := range m.id2servers {
			if sitem.isConnected {
				m.SendMsg(id, msg, &netframe.Server_Extend{
					ServerId: m.MasterConfig.ServerID,
				})
			}
		}
	} else {
		m.SendMsg(serverID, msg, &netframe.Server_Extend{
			ServerId: m.MasterConfig.ServerID,
		})
	}
}

func (m *Master) onSlaveReportLoadLv(ID uint32, serverType uint32, _ uint32, _ []byte, msg interface{}, extend netframe.Server_Extend) {
	req := msg.(*Slave_ReportLoadLevel)

	if req.IsFix {
		if sinfo, ok := m.id2servers[req.BServerID]; ok {
			if sinfo.isConnected && sinfo.isWorking {
				m.routerAddA2B(req.AServerID, req.BServerID)
			} else {
				// 防止A上load值错误
				m.publishLoadLevel(req.AServerID)
			}
		}
	} else {
		m.routerRemoveA2B(req.AServerID, req.BServerID)
	}
}

func (m *Master) printRMsgStat() {
	m.printMsgStatTicker = time.NewTicker(time.Minute * 5)
	go func() {
		for range m.printMsgStatTicker.C {
			if !m.PrintLoadLevelStatus {
				continue
			}
			m.NetIO.Post(func() {
				for aid, sb := range m.routerA2B {
					for bid, item := range sb {
						logger.WithFields(logrus.Fields{
							"asvrid":   aid,
							"bsvrid":   bid,
							"total":    item.msgTotal,
							"5minsCnt": item.msgCnt,
							"avg":      item.msgCnt / 300,
						}).Info("[Router Msg Stat]消息统计")
						item.msgCnt = 0
					}
				}
			})
		}
	}()
}

func (m *Master) addNewSalveCfg(newID uint32) bool {
	//ClusterConfig *ClusterConf
	//// all id->server
	//id2servers map[uint32]*ServerItem
	//// all type->server
	//type2servers map[uint32]*ServerGroup
	if _, ok := m.id2servers[newID]; ok {
		return true
	}

	for _, sconf := range m.ClusterConfig.Slaves {
		if sconf.ServerID == newID {
			var group *ServerGroup
			sitem := &ServerItem{config: sconf, isConnected: false, isWorking: false}
			m.id2servers[sconf.ServerID] = sitem

			// set all type2servers
			if g, ok := m.type2servers[sitem.config.ServerType]; ok {
				group = g
			} else {
				group = &ServerGroup{}
				group.serverType = sitem.config.ServerType
				group.subId2servers = make(map[uint32]*ServerItem)
				group.id2servers = make(map[uint32]*ServerItem)

				m.type2servers[sitem.config.ServerType] = group
			}
			group.id2servers[sitem.config.ServerID] = sitem

			// set subId2servers
			for _, subtype := range sitem.config.SubscribedTypes {
				if group, ok := m.type2servers[subtype]; ok {
					group.subId2servers[sitem.config.ServerID] = sitem
				}
			}

			return true
		}
	}

	return false
}

func (m *Master) onSlavePrepareClose(ID uint32, serverType uint32, _ uint32, _ []byte, msg interface{}, extend netframe.Server_Extend) {
	logger.WithFields(logrus.Fields{
		"svrid":   ID,
		"svrtype": serverType,
	}).Info("Slave Prepare Close.")

	if sitem, ok := m.id2servers[ID]; ok {
		sitem.isWorking = false
		m.publish2MySubscribers(sitem.config.ServerID, sitem.config.ServerType)
	}
}

func (m *Master) onSlaveDisableNewConsistentFLow(ID uint32, serverType uint32, _ uint32, bytes []byte, msg interface{}, extend netframe.Server_Extend) {
	logger.WithFields(logrus.Fields{
		"svrid":   ID,
		"svrtype": serverType,
	}).Info("slave disable new consistent flow")

	if sitem, ok := m.id2servers[ID]; ok {
		sitem.disableNewConsistentFlow = true
		m.publish2MySubscribers(sitem.config.ServerID, sitem.config.ServerType)
	}
}
