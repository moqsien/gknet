package conn

import (
	"bufio"
	"crypto/tls"
	"net"
)

type Context struct {
	Reader     *bufio.Reader
	ReadWriter *bufio.ReadWriter
	RawConn    *Conn
	Conn       net.Conn
}

func (that *Conn) InitContext(tconf *tls.Config, adapter ConnAdapter, callback ...AsyncCallback) (err error) {
	var connection net.Conn = that.Adapt(adapter, callback...)
	if tconf != nil {
		tlsConn := tls.Server(connection, tconf)
		if err = tlsConn.Handshake(); err != nil {
			that.Close()
			return err
		}
		connection = tlsConn
	}
	reader := bufio.NewReader(connection)
	that.Ctx = &Context{
		Reader:     reader,
		ReadWriter: bufio.NewReadWriter(reader, bufio.NewWriter(connection)),
		RawConn:    that,
		Conn:       connection,
	}
	return
}

func (that *Context) Write(data []byte) (int, error) {
	return that.Conn.Write(data)
}

func (that *Context) Read(data []byte) (int, error) {
	return that.Conn.Read(data)
}
