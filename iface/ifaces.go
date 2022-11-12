package iface

import (
	"net"
	"os"
)

type IELoop interface {
	AddConnCount(i int32) int32
	RemoveConn(fd int)
	GetConnList() map[int]net.Conn
	GetConnCount() int32
	GetPoller() IPoller
	StartAsMainLoop(l bool)
	StartAsSubLoop(l bool)
}

type IPoller interface {
	Close() error
	AddPriorTask(f PollTaskFunc, arg PollTaskArg) (err error)
}

type IFd interface {
	GetFd() int
}

type IEngine interface {
	GetOptions() *Options
	GetBalancer() IBalancer
	GetHandler() IEventHandler
}

type IEventHandler interface {
	OnAccept(Conn RawConn) error
	OnOpen(*Context) (data []byte, err error)
	OnTrack(*Context) error
	OnClose(*Context) error
}

type IPollCallback interface {
	Callback(fd int, events uint32) error
	AsyncCallback(fd int, events uint32) chan error
	IsBlocked() bool
}

type IBalancer interface {
	Register(IELoop)
	Next(addr ...net.Addr) IELoop
	Iterator(f BalancerIterFunc)
	Len() int
}

type IListener interface {
	net.Listener
	IFd
	File() (*os.File, error)
	IsUDP() bool
}
