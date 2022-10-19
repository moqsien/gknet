package socket

import (
	"errors"
	"net"
	"strings"
	"syscall"

	"github.com/moqsien/gknet/poll"
)

type IListener interface {
	net.Listener
	poll.IFd
	IsUDP() bool
}

type GkListener struct {
	fd    int
	addr  net.Addr
	isUDP bool
}

func (that *GkListener) Accept() (net.Conn, error) {
	return nil, errors.New("accept not implemented!")
}

func (that *GkListener) Close() (err error) {
	err = syscall.Close(that.fd)
	if err != nil {
		return
	}
	that.fd = -1
	that.addr = nil
	return
}

func (that *GkListener) Addr() net.Addr {
	return that.addr
}

func (that *GkListener) GetFd() int {
	return that.fd
}

func (that *GkListener) IsUDP() bool {
	return that.isUDP
}

func ResolveFd(ln interface{}) (fd int, err error) {
	switch ln.(type) {
	case *net.TCPListener:
		l, _ := ln.(*net.TCPListener)
		file, _ := l.File()
		return int(file.Fd()), nil
	case *net.UnixListener:
		l, _ := ln.(*net.UnixListener)
		file, _ := l.File()
		return int(file.Fd()), nil
	case *net.UDPConn:
		l, _ := ln.(*net.UDPConn)
		file, _ := l.File()
		return int(file.Fd()), nil
	default:
		return -1, errors.New("unsupported Listener")
	}
}

func Listen(network, address string) (gl IListener, err error) {
	if strings.Contains(network, "udp") {
		var addr *net.UDPAddr
		addr, err = net.ResolveUDPAddr(network, address)
		if err != nil {
			return nil, err
		}
		var l *net.UDPConn
		l, err = net.ListenUDP(network, addr)
		if err != nil {
			return nil, err
		}
		var fd int
		fd, err = ResolveFd(l)
		if err != nil {
			return nil, err
		}
		gl = &GkListener{
			fd:    fd,
			addr:  l.LocalAddr(),
			isUDP: true,
		}
		return
	}
	var l net.Listener
	l, err = net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	var fd int
	fd, err = ResolveFd(l)
	if err != nil {
		return nil, err
	}
	gl = &GkListener{
		fd:   fd,
		addr: l.Addr(),
	}
	return
}

func AdaptListener(l net.Listener) (gl IListener, err error) {
	fd, err := ResolveFd(l)
	if err != nil {
		return nil, err
	}
	gl = &GkListener{
		fd:   fd,
		addr: l.Addr(),
	}
	return
}

func AdaptUDPConn(c *net.UDPConn) (gl IListener, err error) {
	fd, err := ResolveFd(c)
	if err != nil {
		return nil, err
	}
	gl = &GkListener{
		fd:    fd,
		addr:  c.LocalAddr(),
		isUDP: true,
	}
	return
}
