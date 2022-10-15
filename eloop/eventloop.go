package eloop

import (
	"bytes"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/moqsien/processes/logger"
	"golang.org/x/sys/unix"

	"github.com/moqsien/gknet/conn"
	"github.com/moqsien/gknet/poll"
	"github.com/moqsien/gknet/socket"
)

type Eloop struct {
	*sync.Mutex
	Listener     net.Listener       // tcp net listener
	ListenerFD   int                // listener file descriptor
	Index        int                // index of worker loop
	Poller       *poll.Poller       // poller
	EventList    []unix.EpollEvent  // events received
	ConnCount    int32              // number of connections
	ConnList     map[int]*conn.Conn // list of connections
	Handler      conn.EventHandler  // Handler for events
	LastIdleTime time.Time          // Last time that number of connections became zero
	Cache        bytes.Buffer       // temporary buffer for scattered bytes
}

func (that *Eloop) AddConnCount(i int32) {
	atomic.AddInt32(&that.ConnCount, i)
}

func (that *Eloop) GetConnCount() int32 {
	return atomic.LoadInt32(&that.ConnCount)
}

// TODO: listener
func (that *Eloop) Accept(_ int, _ uint32) error {
	nfd, sock, err := unix.Accept(that.ListenerFD)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		logger.Errorf("Accept() failed due to error: %v", err)
		return os.NewSyscallError("accept", err)
	}
	if err = os.NewSyscallError("fcntl nonblock", unix.SetNonblock(nfd, true)); err != nil {
		return err
	}

	remoteAddr := socket.SockaddrToTCPOrUnixAddr(sock)
	// if el.engine.opts.TCPKeepAlive > 0 && el.ln.Network == "tcp" {
	// 	err = socket.SetKeepAlivePeriod(nfd, int(el.engine.opts.TCPKeepAlive/time.Second))
	// 	logging.Error(err)
	// }

	c := conn.NewTCPConn(nfd, that.Poller, sock, that.Listener.Addr(), remoteAddr, that.Handler)
	that.Poller.AddRead(c)
	that.ConnList[c.Fd] = c
	return c.Open()
}

func (that *Eloop) RegisterConn(arg poll.PollTaskArg) error {
	c := arg.(*conn.Conn)
	c.Handler = that.Handler
	if err := that.Poller.AddRead(c); err != nil {
		_ = unix.Close(c.Fd)
		return err
	}
	that.ConnList[c.Fd] = c
	return c.Open()
}

func (that *Eloop) CloseAllConn() {
	for _, c := range that.ConnList {
		c.Close()
	}
}

func (that *Eloop) RemoveConn(fd int) {
	delete(that.ConnList, fd)
	that.AddConnCount(-1)
}
