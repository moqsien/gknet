package balancer

import (
	"net"

	"github.com/moqsien/gknet/iface"
)

type RoundRobin struct {
	eloopList []iface.IELoop
	size      int
	nextIndex int
}

func (that *RoundRobin) Len() int { return that.size }

func (that *RoundRobin) Iterator(f iface.BalancerIterFunc) {
	var ok bool
	for key, val := range that.eloopList {
		ok = f(key, val)
		if !ok {
			break
		}
	}
}

func (that *RoundRobin) Register(e iface.IELoop) {
	that.eloopList = append(that.eloopList, e)
	that.size++
}

func (that *RoundRobin) Next(addr ...net.Addr) (e iface.IELoop) {
	e = that.eloopList[that.nextIndex]
	if that.nextIndex++; that.nextIndex > that.size {
		that.nextIndex = 0
	}
	return
}
