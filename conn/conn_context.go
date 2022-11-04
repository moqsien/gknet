package conn

import (
	"bufio"
	"crypto/tls"
	"net"

	"github.com/moqsien/gknet/iface"
)

func (that *Conn) InitContext(tconf *tls.Config, adapter iface.ConnAdapter, callback ...iface.AsyncCallback) (err error) {
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
	that.Ctx = &iface.Context{
		Reader:     reader,
		ReadWriter: bufio.NewReadWriter(reader, bufio.NewWriter(connection)),
		RawConn:    that,
		Conn:       connection,
	}
	return
}
