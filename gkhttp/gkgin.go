package gkhttp

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/moqsien/gknet/iface"
)

type GkGin struct {
	*gin.Engine
	Server  *Server
	options *Opts
}

func NewGin(opts ...*Opts) *GkGin {
	g := &GkGin{Engine: gin.New()}
	if len(opts) > 0 {
		g.options = opts[0]
	}
	return g
}

func (that *GkGin) initServer() {
	if that.Server == nil {
		that.Server = NewHttpServer(that, that.options)
	}
}

func (that *GkGin) Close() {
	if that.Server != nil {
		that.Server.Close()
	}
}

func (that *GkGin) Listen(network, address string) (iface.IListener, error) {
	that.initServer()
	return that.Server.Listen(network, address)
}

func (that *GkGin) AdoptOneListener(ln net.Listener) (iface.IListener, error) {
	that.initServer()
	return that.Server.AdoptOneListener(ln)
}

func (that *GkGin) Serve() error {
	that.initServer()
	return that.Server.Serve()
}

func (that *GkGin) ServeTLS(certFile string, keyFile string) error {
	that.initServer()
	return that.Server.ServeTLS(certFile, keyFile)
}

func (that *GkGin) Run(addr ...string) error {
	that.initServer()
	if len(addr) > 0 {
		if strings.HasSuffix(addr[0], ".sock") {
			that.Listen("unix", addr[0])
		} else {
			that.Listen("tcp", addr[0])
		}
		return that.Serve()
	}
	return that.Serve()
}
