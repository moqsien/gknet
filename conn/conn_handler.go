package conn

import (
	"github.com/moqsien/gknet/iface"
	"github.com/moqsien/gknet/sys"
)

// TODO: Poller.Buffer should be concurrency safe.
func (that *Conn) ReadFromFd() error {
	buf := that.GetBufferFromPool()
	n, err := sys.Read(that.Fd, buf)
	if err != nil || n == 0 {
		if err == sys.EAGAIN {
			return nil
		}
		// conn closed by client.
		if n == 0 {
			err = sys.ECONNRESET
		}
		return that.Close()
	}
	that.Buffer = buf[:n]
	err = that.Handler.OnTrack(that.Ctx)
	that.InBuffer.Write(that.Buffer)
	return err
}

func (that *Conn) WriteToFd() error {
	iov := that.OutBuffer.Peek(-1)
	var (
		n   int
		err error
	)
	if len(iov) > 1 {
		if len(iov) > iface.IovMax {
			iov = iov[:iface.IovMax]
		}
		n, err = sys.Writev(that.Fd, iov)
	} else {
		n, err = sys.Write(that.Fd, iov[0])
	}
	that.OutBuffer.Discard(n)
	switch err {
	case nil:
	case sys.EAGAIN:
		return nil
	default:
		return that.Close()
	}

	if that.OutBuffer.IsEmpty() {
		that.Poller.ModRead(that)
	}
	return nil
}
