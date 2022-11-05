package conn

import (
	"net"

	"github.com/moqsien/gknet/iface"
	"github.com/moqsien/gknet/utils"
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
	if len(data) <= iface.DefaultWritevChunkSize {
		return that.Conn.Write(data)
	}
	return that.Conn.Writev(utils.SplitDataForWritev(data, that.WritevChunkSize))
}

type AsyncWritevConn struct {
	*Conn
	CallBack iface.AsyncCallback
}

func (that *AsyncWritevConn) Write(data []byte) (n int, err error) {
	n = len(data)
	if len(data) <= iface.DefaultWritevChunkSize {
		err = that.Conn.AsyncWrite(data)
	} else {
		err = that.Conn.AsyncWritev(utils.SplitDataForWritev(data, that.WritevChunkSize), that.CallBack)
	}
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
