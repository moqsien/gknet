package conn

import (
	"golang.org/x/sys/unix"

	"github.com/moqsien/gknet/poll"
	"github.com/moqsien/gknet/sys"
	"github.com/moqsien/gknet/utils/errs"
)

/*
writev
*/
func (that *Conn) _writev(fd int, iov [][]byte) (int, error) {
	if len(iov) == 0 {
		return 0, nil
	}
	return sys.Writev(fd, iov)
}

func (that *Conn) _readv(fd int, iov [][]byte) (int, error) {
	if len(iov) == 0 {
		return 0, nil
	}
	return sys.Readv(fd, iov)
}

func (that *Conn) writev(data [][]byte) (n int, err error) {
	for _, d := range data {
		n += len(d)
	}

	if !that.OutBuffer.IsEmpty() {
		_, _ = that.OutBuffer.Writev(data)
		return
	}

	var sent int
	if sent, err = that._writev(that.Fd, data); err != nil {
		if err == unix.EAGAIN {
			that.OutBuffer.Writev(data)
			err = that.Poller.ModReadWrite(that)
			return
		}
		return -1, that.Close()
	}

	if sent < n {
		var pos int
		for i := range data {
			bn := len(data[i])
			if sent < bn {
				data[i] = data[i][sent:]
				pos = i
				break
			}
			sent -= bn
		}
		_, _ = that.OutBuffer.Writev(data[pos:])
		err = that.Poller.ModReadWrite(that)
	}
	return
}

func (that *Conn) asyncWritev(arg poll.PollTaskArg) (err error) {
	if !that.Opened {
		return
	}

	hook := arg.(*AsyncWritevHook)
	_, err = that.writev(hook.Data)
	if hook.Go != nil {
		_ = hook.Go(that)
	}
	return
}

func (that *Conn) AsyncWritev(data [][]byte, cb ...AsyncCallback) error {
	var callback AsyncCallback
	if len(cb) > 0 {
		callback = cb[0]
	}

	if that.IsUDP {
		return errs.ErrUnsupportedOp
	}

	return that.Poller.AddTask(that.asyncWritev, &AsyncWritevHook{
		Go:   callback,
		Data: data,
	})
}
