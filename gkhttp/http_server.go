package gkhttp

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"

	"github.com/moqsien/gknet/engine"
	"github.com/moqsien/gknet/iface"
	"github.com/moqsien/gknet/socket"
	"github.com/moqsien/gknet/utils"
)

const (
	defaultNetwork = "tcp"
	defaultAddr    = "127.0.0.1:8081"
)

type GkEventHandler struct {
	httpServer *Server
}

func (that *GkEventHandler) OnAccept(c iface.RawConn) (err error) {
	return
}

func (that *GkEventHandler) OnOpen(c *iface.Context) (data []byte, err error) {
	return nil, nil
}

func (that *GkEventHandler) OnClose(c *iface.Context) (err error) {
	return
}

func (that *GkEventHandler) OnTrack(c *iface.Context) (err error) {
	if that.httpServer.handler == nil {
		return errors.New("[HttpServer] no handler was found!")
	}
	var req *http.Request
	if that.httpServer.options.DoFast {
		req, err = ReadFastRequest(c.Reader)
	} else {
		req, err = http.ReadRequest(c.Reader)
	}
	if err != nil {
		return err
	}
	res := NewResponse(req, c.Conn, c.ReadWriter)
	that.httpServer.handler.ServeHTTP(res, req)
	res.FinishRequest()
	if that.httpServer.options.DoFast {
		FreeRequest(req)
	}
	FreeResponse(res)
	return nil
}

type Opts struct {
	*iface.Options
	DoFast bool
}

type Server struct {
	handler      http.Handler
	eventHandler iface.IEventHandler
	listener     iface.IListener
	engine       *engine.Engine
	options      *Opts
}

func NewHttpServer(handler http.Handler, opts ...*Opts) *Server {
	s := new(Server)
	s.handler = handler
	s.engine = engine.New()
	s.eventHandler = &GkEventHandler{s}
	if len(opts) > 0 && opts[0] != nil {
		s.options = opts[0]
	}
	return s
}

func (that *Server) GetListener() iface.IListener {
	return that.listener
}

func (that *Server) Close() {
	that.engine.Stop()
}

func (that *Server) Listen(network, address string) (iface.IListener, error) {
	var err error
	that.listener, err = socket.Listen(network, address)
	return that.listener, err
}

func (that *Server) AdoptOneListener(ln net.Listener) (iface.IListener, error) {
	var err error
	that.listener, err = socket.AdaptListener(ln)
	return that.listener, err
}

func (that *Server) Serve() error {
	if that.listener == nil {
		that.Listen(defaultNetwork, defaultAddr)
	}
	if that.options == nil {
		that.options = &Opts{
			Options: &iface.Options{},
		}
	}
	return that.engine.Serve(that.eventHandler, that.listener, that.options.Options)
}

func (that *Server) ServeTLS(certFile, keyFile string) (err error) {
	var config *tls.Config
	if that.options.TLSConfig == nil {
		config = &tls.Config{}
		that.options.TLSConfig = config
	} else {
		config = that.options.TLSConfig
	}

	if !utils.StrSliceContains(config.NextProtos, "http/1.1") {
		config.NextProtos = append(config.NextProtos, "http/1.1")
	}
	configHasCert := len(config.Certificates) > 0 || config.GetCertificate != nil
	if !configHasCert || certFile != "" || keyFile != "" {
		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return
		}
	}
	return that.Serve()
}

// ListenAndServe starts a server, parameter certs orders as certFile, keyFile.
func (that *Server) ListenAndServe(network, address string, certs ...string) (err error) {
	_, err = that.Listen(network, address)
	if err != nil {
		return
	}
	if len(certs) > 1 {
		certFile, keyFile := certs[0], certs[1]
		err = that.ServeTLS(certFile, keyFile)
		return
	}
	err = that.Serve()
	return
}
