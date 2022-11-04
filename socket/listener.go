package socket

import (
	"errors"
	"net"
	"os"
	"strings"

	"github.com/moqsien/gknet/iface"
)

type GkListener struct {
	fd    int
	file  *os.File
	addr  net.Addr
	isUDP bool
}

func (that *GkListener) Accept() (net.Conn, error) {
	return nil, errors.New("Accept not implemented for GkListener!")
}

func (that *GkListener) Close() (err error) {
	err = that.file.Close()
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

func (that *GkListener) File() (*os.File, error) {
	return that.file, nil
}

func (that *GkListener) GetFd() int {
	if that.fd < 0 && that.file != nil {
		that.fd = int(that.file.Fd())
	}
	return that.fd
}

func (that *GkListener) IsUDP() bool {
	return that.isUDP
}

func ResolveFile(ln interface{}) (file *os.File, err error) {
	switch ln.(type) {
	case *net.TCPListener:
		l, _ := ln.(*net.TCPListener)
		return l.File()
	case *net.UnixListener:
		l, _ := ln.(*net.UnixListener)
		return l.File()
	case *net.UDPConn:
		l, _ := ln.(*net.UDPConn)
		return l.File()
	default:
		return nil, errors.New("Unsupported listener!")
	}
}

func Listen(network, address string) (gl iface.IListener, err error) {
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
		var file *os.File
		file, err = ResolveFile(l)
		if err != nil {
			return nil, err
		}
		gl = &GkListener{
			fd:    -1,
			file:  file,
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
	var file *os.File
	file, err = ResolveFile(l)
	if err != nil {
		return nil, err
	}
	gl = &GkListener{
		fd:   -1,
		file: file,
		addr: l.Addr(),
	}
	return
}

func AdaptListener(l net.Listener) (gl iface.IListener, err error) {
	file, err := ResolveFile(l)
	if err != nil {
		return nil, err
	}
	gl = &GkListener{
		fd:   -1,
		file: file,
		addr: l.Addr(),
	}
	return
}

func AdaptUDPConn(c *net.UDPConn) (gl iface.IListener, err error) {
	file, err := ResolveFile(c)
	if err != nil {
		return nil, err
	}
	gl = &GkListener{
		fd:    -1,
		file:  file,
		addr:  c.LocalAddr(),
		isUDP: true,
	}
	return
}
