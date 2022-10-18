package balancer

import (
	"net"

	"github.com/moqsien/gknet/eloop"
)

type LeastConn struct {
	eloopList []*eloop.Eloop
	size      int
}

func (that *LeastConn) Len() int { return that.size }

func (that *LeastConn) Iterator(f eloop.IteratorFunc) {
	var ok bool
	for k, v := range that.eloopList {
		ok = f(k, v)
		if !ok {
			break
		}
	}
}

func (that *LeastConn) Register(e *eloop.Eloop) {
	that.eloopList = append(that.eloopList, e)
	that.size++
}

func (that *LeastConn) Next(addr ...net.Addr) (e *eloop.Eloop) {
	min := that.eloopList[0]
	for _, v := range that.eloopList {
		if v.GetConnCount() < min.GetConnCount() {
			min = v
		}
	}
	return min
}
