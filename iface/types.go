package iface

import (
	"bufio"
	"crypto/tls"
	"net"
	"time"

	"github.com/moqsien/gknet/sys"
)

type BalancerIterFunc func(key int, val IELoop) bool

type Balancer int

type ConnAdapter int

type RawConn interface {
	sys.EventHandler
}

type AsyncCallback func(c net.Conn) error

type AsyncWriteHook struct {
	Go   AsyncCallback
	Data []byte
}

type AsyncWritevHook struct {
	Go   AsyncCallback
	Data [][]byte
}

type Options struct {
	NumOfLoops        int
	LoadBalancer      Balancer
	ReuseAddr         bool
	ReusePort         bool
	SocketWriteBuffer int
	SocketReadBuffer  int
	WriteBuffer       int
	ReadBuffer        int
	ConnKeepAlive     time.Duration
	LockOSThread      bool
	TLSConfig         *tls.Config
	ConnAdapter       ConnAdapter
	ConnAsyncCallback AsyncCallback
}

type Context struct {
	Reader     *bufio.Reader
	ReadWriter *bufio.ReadWriter
	RawConn    RawConn
	Conn       net.Conn
}

func (that *Context) Write(data []byte) (int, error) {
	return that.Conn.Write(data)
}

func (that *Context) Read(data []byte) (int, error) {
	return that.Conn.Read(data)
}

type PollTaskArg interface{}

type PollTaskFunc func(arg PollTaskArg) error
