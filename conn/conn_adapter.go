package conn

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
