package conn

import "net"

type AsyncWriteConn struct {
	*Conn
	CallBack AsyncCallback
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
	CallBack AsyncCallback
}

func (that *AsyncWritevConn) Write(data []byte) (n int, err error) {
	n = len(data)
	// TODO: split data.
	err = that.Conn.AsyncWritev([][]byte{data})
	return
}

type ConnAdapter int

const (
	ConnAsyncWriteAdapter  ConnAdapter = 0
	ConnNoneAdapter        ConnAdapter = 1
	ConnWritevAdapter      ConnAdapter = 2
	ConnAsyncWritevAdapter ConnAdapter = 3
)

// Adapt adapts asyncwrite or writev to net.Conn interface.
func (that *Conn) Adapt(adapter ConnAdapter, callback ...AsyncCallback) net.Conn {
	var cb AsyncCallback = nil
	if len(callback) > 0 {
		cb = callback[0]
	}
	switch adapter {
	case ConnNoneAdapter:
		return that
	case ConnAsyncWriteAdapter:
		return &AsyncWriteConn{Conn: that, CallBack: cb}
	case ConnAsyncWritevAdapter:
		return &AsyncWritevConn{Conn: that, CallBack: cb}
	default:
		return that
	}
}
