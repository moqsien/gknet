package balancer

import (
	"net"

	"github.com/moqsien/gknet/iface"
)

type LeastConn struct {
	eloopList []iface.IELoop
	size      int
}

func (that *LeastConn) Len() int { return that.size }

func (that *LeastConn) Iterator(f iface.BalancerIterFunc) {
	var ok bool
	for k, v := range that.eloopList {
		ok = f(k, v)
		if !ok {
			break
		}
	}
}

func (that *LeastConn) Register(e iface.IELoop) {
	that.eloopList = append(that.eloopList, e)
	that.size++
}

func (that *LeastConn) Next(addr ...net.Addr) (e iface.IELoop) {
	min := that.eloopList[0]
	for _, v := range that.eloopList {
		if v.GetConnCount() < min.GetConnCount() {
			min = v
		}
	}
	return min
}
