package conn

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"

	"github.com/moqsien/processes/logger"
	"github.com/panjf2000/gnet/v2/pkg/buffer/elastic"
	"golang.org/x/sys/unix"

	"github.com/moqsien/gknet/poll"
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

func (that *Conn) open(data []byte) error {
	n, err := unix.Write(that.Fd, data)
	if err != nil && err == unix.EAGAIN {
		_, _ = that.OutBuffer.Write(data)
		return nil
	}

	if err == nil && n < len(data) {
		_, _ = that.OutBuffer.Write(data[n:])
	}

	return err
}

func (that *Conn) write(data []byte) (n int, err error) {
	n = len(data)
	if !that.OutBuffer.IsEmpty() {
		return that.OutBuffer.Write(data)
	}
	var sent int
	if sent, err = unix.Write(that.Fd, data); err != nil {
		if err == unix.EAGAIN {
			_, _ = that.OutBuffer.Write(data)
			err = that.Poller.ModReadWrite(that)
			return
		}
		return -1, that.Close()
	}
	if sent < n {
		_, _ = that.OutBuffer.Write(data[sent:])
		err = that.Poller.ModReadWrite(that)
	}
	return
}

func (that *Conn) sendTo(data []byte) error {
	if that.Sock == nil {
		return unix.Send(that.Fd, data, 0)
	}
	return unix.Sendto(that.Fd, data, 0, that.Sock)
}

func (that *Conn) asyncWrite(arg poll.PollTaskArg) (err error) {
	if !that.Opened {
		return
	}

	hook := arg.(*AsyncWriteHook)
	_, err = that.write(hook.Data)
	if hook.Go != nil {
		hook.Go(that)
	}
	return
}

/*
public methods
*/
func (that *Conn) GetFd() int {
	return that.Fd
}

func (that *Conn) Close() (rerr error) {
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
			if n, e := that._writev(that.Fd, iov); e != nil {
				logger.Warningf("closeConn: error occurs when sending data back to peer, %v", e)
				break
			} else {
				that.OutBuffer.Discard(n)
			}
		}
	}

	err0, err1 := that.Poller.RemoveFd(that), unix.Close(that.Fd)
	if err0 != nil {
		rerr = fmt.Errorf("failed to delete fd=%d from poller: %v", that.Fd, err0)
	}
	if err1 != nil {
		err1 = fmt.Errorf("failed to close fd=%d: %v", that.Fd, os.NewSyscallError("close", err1))
		if rerr != nil {
			rerr = errors.New(rerr.Error() + " & " + err1.Error())
		} else {
			rerr = err1
		}
	}
	that.Poller.Eloop.RemoveConn(that.Fd)
	// if el.eventHandler.OnClose(c, err) == Shutdown {
	// 	rerr = gerrors.ErrEngineShutdown
	// }
	that.releaseTCP()
	return
}

func (that *Conn) Open() error {
	that.Opened = true
	var err error
	data, _ := that.Handler.OnOpen(that)
	if data != nil {
		if err = that.open(data); err != nil {
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

/*
async apis
*/
func (that *Conn) AsyncWrite(data []byte, cb ...AsyncCallback) error {
	var callback AsyncCallback
	if len(cb) > 0 {
		callback = cb[0]
	}

	if that.IsUDP {
		defer func() {
			if callback != nil {
				cb[0](that)
			}
		}()
		return that.sendTo(data)
	}

	return that.Poller.AddTask(that.asyncWrite, &AsyncWriteHook{
		Go:   callback,
		Data: data,
	})
}

/*
public methods for poller callback
*/
func (that *Conn) ReadFromFd() error {
	n, err := unix.Read(that.Fd, that.Buffer)
	if err != nil || n == 0 {
		if err == unix.EAGAIN {
			return nil
		}
		if n == 0 {
			err = unix.ECONNRESET
		}
		return that.Close()
	}
	that.Handler.OnTrack(that)
	that.InBuffer.Write(that.Buffer[:n])
	return nil
}

func (that *Conn) WriteToFd() error {
	iov := that.OutBuffer.Peek(-1)
	var (
		n   int
		err error
	)
	if len(iov) > 1 {
		if len(iov) > IovMax {
			iov = iov[:IovMax]
		}
		n, err = that._writev(that.Fd, iov)
	} else {
		n, err = unix.Write(that.Fd, iov[0])
	}
	that.OutBuffer.Discard(n)
	switch err {
	case nil:
	case unix.EAGAIN:
		return nil
	default:
		return that.Close()
	}

	// All data have been drained, it's no need to monitor the writable events,
	// remove the writable event from poller to help the future event-loops.
	if that.OutBuffer.IsEmpty() {
		that.Poller.ModRead(that)
	}
	return nil
}
