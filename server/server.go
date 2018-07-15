package server

import (
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/juju/loggo"
	"github.com/lzjluzijie/websocks/core"
)

//todo
var logger = loggo.GetLogger("server")

type WebSocksServer struct {
	*Config
	LogLevel loggo.Level

	Upgrader   *websocket.Upgrader
	muxConnMap sync.Map
	mutex      sync.Mutex

	CreatedAt time.Time
	Stats     *core.Stats
}

func (server *WebSocksServer) HandleWebSocket(c *gin.Context) {
	w := c.Writer
	r := c.Request
	wsConn, err := server.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Debugf(err.Error())
		return
	}
	defer wsConn.Close()

	ws := core.NewWebSocket(wsConn, server.Stats)
	//todo conns

	////mux
	//if r.Header.Get("WebSocks-Mux") == "mux" {
	//	muxWS := NewMuxWebSocket(ws)
	//	muxWS.ServerListen()
	//	return
	//}

	host := r.Header.Get("WebSocks-Host")
	logger.Debugf("Dial %s", host)
	conn, err := server.DialRemote(host)
	if err != nil {
		logger.Debugf(err.Error())
		return
	}

	go func() {
		_, err = io.Copy(conn, ws)
		if err != nil {
			logger.Debugf(err.Error())
			return
		}
	}()

	_, err = io.Copy(ws, conn)
	if err != nil {
		logger.Debugf(err.Error())
		return
	}

	return
}

func (server *WebSocksServer) DialRemote(host string) (conn net.Conn, err error) {
	conn, err = net.Dial("tcp", host)
	if err != nil {
		return
	}
	return
}

func (server *WebSocksServer) Run() (err error) {
	r := gin.Default()
	r.GET(server.Pattern, server.HandleWebSocket)

	if server.ReverseProxy != "" {
		remote, err := url.Parse(server.ReverseProxy)
		if err != nil {
			panic(err)
		}
		proxy := httputil.NewSingleHostReverseProxy(remote)
		r.NoRoute(func(c *gin.Context) {
			proxy.ServeHTTP(c.Writer, c.Request)
		})
	}

	s := http.Server{
		Addr:    server.ListenAddr,
		Handler: r,
	}

	logger.Infof("Start to listen at %s", server.ListenAddr)

	if !server.TLS {
		err = s.ListenAndServe()
		if err != nil {
			return err
		}
		return
	}
	return
}
