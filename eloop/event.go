/*
EloopEvent defines the event type for Eloop. An acception or connection monitoring.
*/
package eloop

import (
	"errors"
	"sync"

	"github.com/moqsien/processes/logger"

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

func (that *EloopEventAccept) AsyncCallback(fd int, events uint32) (errChan chan error) {
	logger.Println("Not implemented!")
	return
}

func (that *EloopEventAccept) AsyncWaitCallback(fd int, events uint32, wg *sync.WaitGroup) (errChan chan error) {
	logger.Println("Not implemented!")
	return
}

type EloopEventHandleConn struct {
	Eloop *Eloop
}

func (that *EloopEventHandleConn) IsBlocked() bool { return false }

var pollEvBufffer = make([]byte, 128)

func (that *EloopEventHandleConn) Callback(fd int, events uint32) error {
	if fd == that.Eloop.Poller.GetPollEvFd() || fd == 0 {
		sys.Read(fd, pollEvBufffer)
		return nil
	}
	if connection, found := that.Eloop.ConnList[fd]; found {
		return sys.HandleEvents(events, connection.(*conn.Conn))
	}
	return errors.New("Connection not found!")
}

func (that *EloopEventHandleConn) AsyncCallback(fd int, events uint32) (errChan chan error) {
	if fd == that.Eloop.Poller.GetPollEvFd() || fd == 0 {
		sys.Read(fd, pollEvBufffer)
		return
	}
	if connection, found := that.Eloop.ConnList[fd]; found {
		c := connection.(*conn.Conn)
		sys.AsyncHandleEvents(events, c)
		return c.ErrChan
	}
	logger.Warningf("Fd not found: %d", fd)
	return
}

func (that *EloopEventHandleConn) AsyncWaitCallback(fd int, events uint32, wg *sync.WaitGroup) (errChan chan error) {
	if fd == that.Eloop.Poller.GetPollEvFd() || fd == 0 {
		sys.Read(fd, pollEvBufffer)
		return
	}
	if connection, found := that.Eloop.ConnList[fd]; found {
		c := connection.(*conn.Conn)
		sys.AsyncHandleEventsAndWait(events, c, wg)
		return c.ErrChan
	}
	logger.Warningf("Fd not found: %d", fd)
	return
}
