package balancer

import (
	"net"

	"github.com/moqsien/gknet/eloop"
)

type RoundRobin struct {
	eloopList []*eloop.Eloop
	size      int
	nextIndex int
}

func (that *RoundRobin) Len() int { return that.size }

func (that *RoundRobin) Iterator(f eloop.IteratorFunc) {
	var ok bool
	for key, val := range that.eloopList {
		ok = f(key, val)
		if !ok {
			break
		}
	}
}

func (that *RoundRobin) Register(e *eloop.Eloop) {
	that.eloopList = append(that.eloopList, e)
	that.size++
}

func (that *RoundRobin) Next(addr ...net.Addr) (e *eloop.Eloop) {
	e = that.eloopList[that.nextIndex]
	if that.nextIndex++; that.nextIndex > that.size {
		that.nextIndex = 0
	}
	return
}
