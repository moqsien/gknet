package gkhttp

import (
	"crypto/tls"
	"net/http"

	"github.com/moqsien/gknet/engine"
	"github.com/moqsien/gknet/iface"
)

type GkEventHandler struct{}

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
	return
}

type Server struct {
	handler      http.Handler
	eventHandler iface.IEventHandler
	listener     iface.IListener
	engine       *engine.Engine
	tlsConf      *tls.Config
	options      *iface.Options
}
