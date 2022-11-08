package eloop

import (
	"errors"

	"github.com/moqsien/gknet/conn"
	"github.com/moqsien/gknet/sys"
)

type EloopEventAccept struct {
	Eloop *Eloop
}

func (that *EloopEventAccept) IsBlocked() bool { return true }

func (that *EloopEventAccept) Callback(fd int, events uint32) error {
	return that.Eloop.Accept(fd, events)
}

type EloopEventHandleConn struct {
	Eloop *Eloop
}

func (that *EloopEventHandleConn) IsBlocked() bool { return false }

func (that *EloopEventHandleConn) Callback(fd int, events uint32) error {
	if connection, found := that.Eloop.ConnList[fd]; found {
		return sys.HandleEvents(events, connection.(*conn.Conn))
	}
	return errors.New("Connection not found!")
}