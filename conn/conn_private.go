package conn

import (
	"syscall"

	"github.com/moqsien/gknet/poll"
	"github.com/moqsien/gknet/sys"
)

func (that *Conn) open(data []byte) error {
	n, err := sys.Write(that.Fd, data)
	if err != nil && err == sys.EAGAIN {
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
	if sent, err = sys.Write(that.Fd, data); err != nil {
		if err == sys.EAGAIN {
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
	return syscall.Sendto(that.Fd, data, 0, that.Sock)
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
