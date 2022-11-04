package conn

import (
	"github.com/moqsien/gknet/iface"
	"github.com/moqsien/gknet/sys"
	"github.com/moqsien/gknet/utils/errs"
)

func (that *Conn) writeUdp(data []byte) error {
	return sys.WriteUdp(that.Fd, data, 0, that.Sock)
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

func (that *Conn) writeOnOpen(data []byte) (n int, err error) {
	return that.write(data)
}

func (that *Conn) asyncWrite(arg iface.PollTaskArg) (err error) {
	if !that.Opened {
		return
	}

	hook, ok := arg.(*iface.AsyncWriteHook)
	if ok {
		_, err = that.write(hook.Data)
		if hook.Go != nil {
			hook.Go(that)
		}
	}
	return
}

func (that *Conn) Write(p []byte) (int, error) {
	if that.IsUDP {
		if err := that.writeUdp(p); err != nil {
			return 0, err
		}
		return len(p), nil
	}
	return that.write(p)
}

func (that *Conn) AsyncWrite(data []byte, cb ...iface.AsyncCallback) error {
	var callback iface.AsyncCallback
	if len(cb) > 0 {
		callback = cb[0]
	}

	if that.IsUDP {
		defer func() {
			if callback != nil {
				cb[0](that)
			}
		}()
		return that.writeUdp(data)
	}

	return that.Poller.AddTask(that.asyncWrite, &iface.AsyncWriteHook{
		Go:   callback,
		Data: data,
	})
}

func (that *Conn) writev(data [][]byte) (n int, err error) {
	for _, b := range data {
		n += len(b)
	}

	if !that.OutBuffer.IsEmpty() {
		_, _ = that.OutBuffer.Writev(data)
		return
	}

	var sent int
	if sent, err = sys.Writev(that.Fd, data); err != nil {
		if err == sys.EAGAIN {
			_, _ = that.OutBuffer.Writev(data)
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

func (that *Conn) asyncWritev(arg iface.PollTaskArg) (err error) {
	if !that.Opened {
		return nil
	}

	hook := arg.(*iface.AsyncWritevHook)
	_, err = that.writev(hook.Data)
	if hook.Go != nil {
		err = hook.Go(that)
	}
	return
}

func (that *Conn) Writev(bs [][]byte) (int, error) {
	if that.IsUDP {
		return 0, errs.ErrUnsupportedOp
	}
	return that.writev(bs)
}

func (that *Conn) AsyncWritev(bs [][]byte, cb ...iface.AsyncCallback) error {
	var callback iface.AsyncCallback
	if len(cb) > 0 {
		callback = cb[0]
	}
	if that.IsUDP {
		return errs.ErrUnsupportedOp
	}
	return that.Poller.AddTask(that.asyncWritev, &iface.AsyncWritevHook{Go: callback, Data: bs})
}
