package websocket

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/net/netutil"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"github/beijian128/micius/frame/network/crypto"
	"github/beijian128/micius/frame/network/msgpackager"
	"github/beijian128/micius/frame/network/msgprocessor"
)

var logger = logrus.WithField("module", "websocket")

const serverMaxConnCnt = 100000

// Server webs
type Server struct {
	wg           sync.WaitGroup
	msgPackager  msgpackager.MsgPackager
	msgProcessor msgprocessor.MsgProcessor
	newCrypto    func() crypto.Crypto

	upgrader websocket.Upgrader
	httpSvr  *http.Server
}

// NewServer 创建 websocket server, 绑定回调函数
func NewServer(
	name string,
	addr string,
	maxConnCnt int,
	writeChanLen int,
	closeBuffFull bool,
	msgPackager msgpackager.MsgPackager,
	msgProcessor msgprocessor.MsgProcessor,
	newCrypto func() crypto.Crypto,
	disableCheckOrigin bool,
	openTLS bool, //是否开启TLS
	certFile string,
	keyFile string,
) (*Server, error) {
	s := new(Server)
	s.msgPackager = msgPackager
	s.msgProcessor = msgProcessor
	s.newCrypto = newCrypto

	if disableCheckOrigin {
		s.upgrader.CheckOrigin = func(*http.Request) bool {
			return true
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		c, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.WithError(err).Warn("Websocket upgrader falied")
			return
		}

		conn := newConnection(c, true, writeChanLen, closeBuffFull, msgPackager, msgProcessor, newCrypto())
		conn.readLoop()
	})

	s.httpSvr = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	if addr == "" {
		addr = ":http"
		if openTLS {
			addr = "https"
		}
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	if maxConnCnt <= 0 {
		maxConnCnt = serverMaxConnCnt
	}
	ln = netutil.LimitListener(ln, maxConnCnt)

	go func() {
		var err error
		if openTLS {
			err = s.httpSvr.ServeTLS(ln, certFile, keyFile)
		} else {
			err = s.httpSvr.Serve(ln)
		}

		if err != nil && err != http.ErrServerClosed {
			logger.WithError(err).WithField("addr", addr).Fatal("Http Serve failed") //Fatal会退出程序
		} else {
			logger.WithField("addr", addr).Info("Http server closed")
		}
	}()

	return s, nil
}

// Close 停止监听并关闭所有连接
func (s *Server) Close() {
	s.httpSvr.Close()
}
