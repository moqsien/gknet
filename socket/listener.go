package socket

import (
	"errors"
	"net"
	"strings"
)

type UDPListener struct {
	*net.UDPConn
}

func (that *UDPListener) Accept() (net.Conn, error) {
	return nil, errors.New("udp accept is not supported!")
}

func (that *UDPListener) Addr() net.Addr {
	return that.UDPConn.LocalAddr()
}

func (that *UDPListener) Close() error {
	return that.UDPConn.Close()
}

type NetListener interface {
	net.Listener
	GetFd() int
}

type Listener struct {
	net.Listener
	Fd int
}

func (that *Listener) GetFd() int {
	if that.Fd != 0 {
		return that.Fd
	}
	switch that.Listener.(type) {
	case *net.TCPListener:
		l := that.Listener.(*net.TCPListener)
		if file, err := l.File(); err == nil {
			that.Fd = int(file.Fd())
		}
	case *net.UnixListener:
		l := that.Listener.(*net.UnixListener)
		if file, err := l.File(); err == nil {
			that.Fd = int(file.Fd())
		}
	case *UDPListener:
		l := that.Listener.(*UDPListener)
		if file, err := l.File(); err == nil {
			that.Fd = int(file.Fd())
		}
	default:
	}
	return that.Fd
}

func NewListener(network, address string) NetListener {
	if strings.Contains(network, "udp") {
		addr, err := net.ResolveUDPAddr(network, address)
		if err != nil {
			panic(err)
		}
		l, err := net.ListenUDP(network, addr)
		if err != nil {
			panic(err)
		}
		return &Listener{
			Listener: &UDPListener{l},
		}
	}
	l, err := net.Listen(network, address)
	if err != nil {
		panic(err)
	}
	return &Listener{
		Listener: l,
	}
}
