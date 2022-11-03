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

func (that *Conn) InitContext(tconf *tls.Config) (err error) {
	var connection net.Conn = that
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
