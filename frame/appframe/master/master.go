package master

import (
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/sirupsen/logrus"

	"github/beijian128/micius/frame/framework/netcluster"
	"github/beijian128/micius/frame/ioservice"
	"github/beijian128/micius/frame/log"
)

// DisableMasterInitGlobalLogrus 创建 Master 时, 禁止初始化全局的 logrus 设置.
// 在有多个 Application 实例时, 该选项应该设置为 true. (例如 All in One 模式)
var DisableMasterInitGlobalLogrus bool

// Master 服务.
type Master struct {
	impl   *netcluster.Master
	worker worker.Worker
	name   string
	id     uint32

	closeLogger func()
}

// New 创建 Master 服务.
func New(netConfigFile string, name string) (*Master, error) {
	netConfig, err := netcluster.ParseClusterConfigFile(netConfigFile)
	if err != nil {
		return nil, err
	}
	cfg, ok := netConfig.Masters[name]
	if !ok {
		return nil, fmt.Errorf("can not find master config by name %s", name)
	}

	worker := worker.NewWorker(fmt.Sprintf("app_%s_main", name), 102400)
	worker.Init()

	m := new(Master)
	m.name = name
	m.id = cfg.ServerID
	m.worker = worker
	m.impl = netcluster.NewMaster(netConfig, name, worker)
	if m.impl == nil {
		return nil, errors.New("new master")
	}
	m.impl.Init()

	if !DisableMasterInitGlobalLogrus {
		close, err := log.InitLogrus(&cfg.Log)
		if err != nil {
			return nil, err
		}
		m.closeLogger = close
	}
	return m, nil
}

// Run 运行 Master, 阻塞调用.
// 退出使用信号 os.Interrupt 或 os.Kill.
func (m *Master) Run() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	m.worker.Run()
	m.impl.Run()

	logrus.WithField("id", m.id).Infof("Master (%s) running...", m.name)

	sig := <-c
	logrus.WithField("id", m.id).Infof("Master (%s) exiting... signal:(%v)", m.name, sig)

	m.impl.Fini()
	m.worker.Fini()

	if m.closeLogger != nil {
		m.closeLogger()
	}
}
