package eloop

import (
	"bytes"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/moqsien/gknet/conn"
	"github.com/moqsien/gknet/poll"
	"github.com/moqsien/gknet/socket"
	"github.com/moqsien/gknet/sys"
)

type Eloop struct {
	Listener     socket.IListener   // net listener
	Index        int                // index of worker loop
	Poller       *poll.Poller       // poller
	ConnCount    int32              // number of connections
	ConnList     map[int]*conn.Conn // list of connections
	Handler      conn.EventHandler  // Handler for events
	LastIdleTime time.Time          // Last time that number of connections became zero
	Cache        bytes.Buffer       // temporary buffer for scattered bytes
	Balancer     IBalancer          // load balancer
	TcpTimeout   int                // how many seconds does a tcp connection keep alive
}

func (that *Eloop) RegisterConn(arg poll.PollTaskArg) error {
	c := arg.(*conn.Conn)
	c.Handler = that.Handler
	var err error
	if err = that.Poller.AddRead(c); err != nil {
		_ = syscall.Close(c.Fd)
		return err
	}
	that.ConnList[c.Fd] = c
	err = c.Open()
	if err == nil {
		that.AddConnCount(1)
	}
	return err
}

func (that *Eloop) packTcpConn(nfd int, sock syscall.Sockaddr) {
	remoteAddr := socket.SockaddrToTCPOrUnixAddr(sock)
	c := conn.NewTCPConn(nfd, that.Poller, sock, that.Listener.Addr(), remoteAddr, that.Handler)
	loop := that.Balancer.Next(c.AddrLocal)
	that.Poller.AddPriorTask(loop.RegisterConn, c)
}

func (that *Eloop) Accept(_ int, _ uint32) error {
	nfd, sock, err := sys.Accept(that.Listener.GetFd(), that.TcpTimeout)
	if err != nil {
		return err
	}
	that.packTcpConn(nfd, sock)
	return err
}

func (that *Eloop) ActivateMainLoop(l bool) {
	if l {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	that.Poller.AddRead(that.Listener)
	that.Poller.Start(func(fd int, events uint32) error {
		return that.Accept(fd, events)
	})
}

func (that *Eloop) ActivateSubLoop(l bool) {
	if l {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	that.Poller.Start(func(fd int, events uint32) error {
		if conn, found := that.ConnList[fd]; found {
			sys.HandleEvents(events, conn)
		}
		return nil
	})
}

func (that *Eloop) AddConnCount(i int32) {
	atomic.AddInt32(&that.ConnCount, i)
}

func (that *Eloop) GetConnCount() int32 {
	return atomic.LoadInt32(&that.ConnCount)
}

func (that *Eloop) RemoveConn(fd int) {
	delete(that.ConnList, fd)
	that.AddConnCount(-1)
}

func (that *Eloop) CloseAllConn() {
	for _, c := range that.ConnList {
		c.Close()
	}
}
