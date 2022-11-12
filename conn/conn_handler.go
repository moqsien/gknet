package conn

import (
	"github.com/moqsien/processes/logger"

	"github.com/moqsien/gknet/iface"
	"github.com/moqsien/gknet/sys"
	"github.com/moqsien/gknet/utils/errs"
)

func (that *Conn) sendErr(err error) {
	switch err {
	case nil:
		return
	case errs.ErrEngineShutdown, errs.ErrAcceptSocket:
		select {
		case that.ErrChan <- err:
			return
		default:
			return
		}
	default:
		logger.Println("[AsyncHandleEvents]", err)
	}
}

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
	that.PutBufferToPool(buf)
	return err
}

func (that *Conn) AsyncReadFromFd() {
	that.Poller.Pool.Submit(func() {
		that.lock.Lock()
		defer that.lock.Unlock()
		buf := that.GetBufferFromPool()
		n, err := sys.Read(that.Fd, buf)
		if err != nil || n == 0 {
			if err == sys.EAGAIN {
				return
			}
			// conn closed by client.
			if n == 0 {
				err = sys.ECONNRESET
			}
			err = that.Close()
			that.sendErr(err)
			return
		}
		that.Buffer = buf[:n]
		err = that.Handler.OnTrack(that.Ctx)
		that.InBuffer.Write(that.Buffer)
		that.PutBufferToPool(buf)
		that.sendErr(err)
	})
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

func (that *Conn) AsyncWriteToFd() {
	that.Poller.Pool.Submit(func() {
		that.lock.Lock()
		defer that.lock.Unlock()
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
			return
		default:
			err = that.Close()
			that.sendErr(err)
			return
		}

		if that.OutBuffer.IsEmpty() {
			that.Poller.ModRead(that)
		}
		return
	})
}

func (that *Conn) AsyncClose() {
	that.lock.Lock()
	defer that.lock.Unlock()
	err := that.Close()
	that.sendErr(err)
}
