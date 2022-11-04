package conn

import (
	"net"

	"github.com/moqsien/gknet/iface"
)

type AsyncWriteConn struct {
	*Conn
	CallBack iface.AsyncCallback
}

func (that *AsyncWriteConn) Write(data []byte) (n int, err error) {
	n = len(data)
	err = that.Conn.AsyncWrite(data, that.CallBack)
	return
}

type WritevConn struct {
	*Conn
}

func (that *WritevConn) Write(data []byte) (n int, err error) {
	// TODO: split data.
	return that.Conn.Writev([][]byte{data})
}

type AsyncWritevConn struct {
	*Conn
	CallBack iface.AsyncCallback
}

func (that *AsyncWritevConn) Write(data []byte) (n int, err error) {
	n = len(data)
	// TODO: split data.
	err = that.Conn.AsyncWritev([][]byte{data}, that.CallBack)
	return
}

// Adapt adapts asyncwrite or writev to net.Conn interface.
func (that *Conn) Adapt(adapter iface.ConnAdapter, callback ...iface.AsyncCallback) net.Conn {
	var cb iface.AsyncCallback = nil
	if len(callback) > 0 {
		cb = callback[0]
	}
	switch adapter {
	case iface.ConnNoneAdapter:
		return that
	case iface.ConnAsyncWriteAdapter:
		return &AsyncWriteConn{Conn: that, CallBack: cb}
	case iface.ConnAsyncWritevAdapter:
		return &AsyncWritevConn{Conn: that, CallBack: cb}
	default:
		return that
	}
}
