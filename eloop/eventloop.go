package eloop

import (
	"net"
	"runtime"
	"sync/atomic"
	"syscall"

	"github.com/moqsien/gknet/conn"
	"github.com/moqsien/gknet/iface"
	"github.com/moqsien/gknet/poll"
	"github.com/moqsien/gknet/socket"
	"github.com/moqsien/gknet/sys"
	"github.com/moqsien/gknet/utils/errs"
)

type Eloop struct {
	Listener  iface.IListener  // net listener
	Index     int              // index of worker loop
	Poller    *poll.Poller     // poller
	Engine    iface.IEngine    // engine
	Balancer  iface.IBalancer  // balancer
	ConnCount int32            // number of connections
	ConnList  map[int]net.Conn // list of connections
}

func (that *Eloop) RegisterConn(arg iface.PollTaskArg) error {
	c := arg.(*conn.Conn)
	var err error
	if err = c.Poller.AddRead(c); err != nil {
		_ = syscall.Close(c.Fd)
		return err
	}
	// tls handshaking and context preparation.
	err = c.InitContext(that.Engine.GetOptions().TLSConfig,
		that.Engine.GetOptions().ConnAdapter,
		that.Engine.GetOptions().ConnAsyncCallback)
	that.ConnList[c.Fd] = c
	err = c.Open()
	if err == nil {
		that.ConnCount = that.AddConnCount(1)
	}
	return err
}

func (that *Eloop) chooseEloop(addrLocal net.Addr) iface.IELoop {
	if that.Balancer == nil {
		that.Balancer = that.Engine.GetBalancer()
	}
	return that.Balancer.Next(addrLocal)
}

func (that *Eloop) packTcpConn(nfd int, sock syscall.Sockaddr) (c *conn.Conn) {
	remoteAddr := socket.SockaddrToTCPOrUnixAddr(sock)
	c = conn.NewTCPConn(nfd)
	c.SetConn(&conn.ConnOpts{
		SockAddr:          sock,
		LocalAddr:         that.Listener.Addr(),
		RemoteAddr:        remoteAddr,
		Handler:           that.Engine.GetHandler(),
		WriteBufferCap:    that.Engine.GetOptions().WriteBuffer,
		WritevChunkSize:   that.Engine.GetOptions().WritevChunkSize,
		SocketWriteBuffer: that.Engine.GetOptions().SocketWriteBuffer,
		SocketReadBuffer:  that.Engine.GetOptions().SocketReadBuffer,
	})
	loop := that.chooseEloop(c.AddrLocal).(*Eloop)
	c.Poller = loop.Poller
	that.Poller.AddPriorTask(loop.RegisterConn, c)
	return
}

func (that *Eloop) Accept(_ int, _ uint32) error {
	nfd, sock, err := sys.Accept(that.Listener.GetFd(), that.Engine.GetOptions().ConnKeepAlive)
	if err != nil {
		return errs.ErrAcceptSocket
	}
	c := that.packTcpConn(nfd, sock)
	err = that.Engine.GetHandler().OnAccept(c)
	return err
}

func (that *Eloop) StartAsMainLoop(l bool) {
	if l {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	that.Poller.AddRead(that.Listener)
	that.Poller.Start(&EloopEventAccept{Eloop: that})
}

func (that *Eloop) StartAsSubLoop(l bool) {
	if l {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	that.Poller.Start(&EloopEventHandleConn{Eloop: that})
}

func (that *Eloop) AddConnCount(i int32) int32 {
	return atomic.AddInt32(&that.ConnCount, i)
}

func (that *Eloop) GetConnCount() int32 {
	return atomic.LoadInt32(&that.ConnCount)
}

func (that *Eloop) RemoveConn(fd int) {
	delete(that.ConnList, fd)
	that.ConnCount = that.AddConnCount(-1)
}

func (that *Eloop) CloseAllConn() {
	for _, c := range that.ConnList {
		c.Close()
	}
}

func (that *Eloop) GetConnList() map[int]net.Conn {
	return that.ConnList
}

func (that *Eloop) GetPoller() iface.IPoller {
	return that.Poller
}
