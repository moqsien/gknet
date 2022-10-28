package conn

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"

	"github.com/moqsien/processes/logger"
	"github.com/panjf2000/gnet/v2/pkg/buffer/elastic"

	"github.com/moqsien/gknet/poll"
	"github.com/moqsien/gknet/sys"
	"github.com/moqsien/gknet/utils/errs"
)

const (
	IovMax = 1024
)

type Conn struct {
	Fd         int
	Poller     *poll.Poller
	Sock       syscall.Sockaddr
	AddrLocal  net.Addr
	AddrRemote net.Addr
	OutBuffer  *elastic.Buffer
	InBuffer   elastic.RingBuffer
	Buffer     []byte
	IsUDP      bool
	Ctx        interface{}
	Opened     bool
	Handler    EventHandler
}

// new Conn
func NewTCPConn(fd int, poller *poll.Poller, sa syscall.Sockaddr, localAddr, remoteAddr net.Addr, h EventHandler) (c *Conn) {
	c = &Conn{
		Fd:         fd,
		Sock:       sa,
		Poller:     poller,
		AddrLocal:  localAddr,
		AddrRemote: remoteAddr,
		Handler:    h,
	}
	c.OutBuffer, _ = elastic.New(1024)
	return
}

/*
private methods
*/
func (that *Conn) releaseUDP() {
	that.Ctx = nil
	that.AddrLocal = nil
	that.AddrRemote = nil
	that.Buffer = nil
}

func (that *Conn) releaseTCP() {
	that.Ctx = nil
	that.Opened = false
	that.Sock = nil
	that.Buffer = nil
	that.AddrLocal = nil
	that.AddrRemote = nil
	that.InBuffer.Done()
	that.OutBuffer.Release()
}

/*
public methods
*/
func (that *Conn) GetFd() int {
	return that.Fd
}

func (that *Conn) Close(err ...syscall.Errno) (rerr error) {
	if addr := that.AddrLocal; addr != nil && strings.HasPrefix(that.AddrLocal.Network(), "udp") {
		that.releaseUDP()
		return
	}

	if !that.Opened {
		return
	}

	if !that.OutBuffer.IsEmpty() {
		for !that.OutBuffer.IsEmpty() {
			iov := that.OutBuffer.Peek(0)
			if len(iov) > IovMax {
				iov = iov[:IovMax]
			}
			if n, e := sys.Writev(that.Fd, iov); e != nil {
				logger.Warningf("closeConn: error occurs when sending data back to peer, %v", e)
				break
			} else {
				that.OutBuffer.Discard(n)
			}
		}
	}

	err0, err1 := that.Poller.RemoveFd(that), sys.CloseFd(that.Fd)
	if err0 != nil {
		rerr = fmt.Errorf("failed to delete fd=%d from poller: %v", that.Fd, err0)
	}
	if err1 != nil {
		err1 = fmt.Errorf("failed to close fd=%d: %v", that.Fd, err1)
		if rerr != nil {
			rerr = errors.New(rerr.Error() + " & " + err1.Error())
		} else {
			rerr = err1
		}
	}
	that.Poller.Eloop.RemoveConn(that.Fd)
	var eee error
	if len(err) > 0 {
		eee = err[0]
	}
	if that.Handler.OnClose(that, eee) != nil {
		rerr = errs.ErrEngineShutdown
	}
	that.releaseTCP()
	return
}

func (that *Conn) Open() error {
	that.Opened = true
	var err error
	data, _ := that.Handler.OnOpen(that)
	if data != nil {
		if _, err = that.writeOnOpen(data); err != nil {
			return err
		}
	}

	if !that.OutBuffer.IsEmpty() {
		if err = that.Poller.AddWrite(that); err != nil {
			return err
		}
	}
	return err
}
