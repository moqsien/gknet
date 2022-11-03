package eloop

import (
	"net"

	"github.com/moqsien/gknet/conn"
)

type IteratorFunc func(key int, val *Eloop) bool

type IBalancer interface {
	Register(*Eloop)
	Next(addr ...net.Addr) *Eloop
	Iterator(f IteratorFunc)
	Len() int
}

type IEngine interface {
	GetOptions() *Options
	GetBalancer() IBalancer
	GetHandler() conn.IEventHandler
}
