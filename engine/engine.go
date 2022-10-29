package engine

import (
	"sync"

	"github.com/moqsien/gknet/conn"
	"github.com/moqsien/gknet/eloop"
	"github.com/moqsien/gknet/socket"
)

type Engine struct {
	Ln        socket.IListener
	Balancer  eloop.IBalancer
	MainLoop  *eloop.Eloop
	Handler   conn.EventHandler
	IsClosing int32
	wg        sync.WaitGroup
	cond      *sync.Cond
	once      sync.Once
}
