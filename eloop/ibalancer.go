package eloop

import "net"

type IteratorFunc func(key int, val *Eloop) bool

type IBalancer interface {
	Register(*Eloop)
	Next(addr ...net.Addr) *Eloop
	Iterator(f IteratorFunc)
	Len() int
}
