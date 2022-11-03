package eloop

import (
	"crypto/tls"
	"time"

	"github.com/moqsien/gknet/conn"
)

const (
	RoundRobinLB       int = 0
	LeastConnLB        int = 1
	MaxStreamBufferCap int = 64 * 1024
)

type Options struct {
	NumOfLoops        int
	LoadBalancer      int
	ReuseAddr         bool
	ReusePort         bool
	SocketWriteBuffer int
	SocketReadBuffer  int
	WriteBuffer       int
	ReadBuffer        int
	ConnKeepAlive     time.Duration
	LockOSThread      bool
	TLSConfig         *tls.Config
	ConnAdapter       conn.ConnAdapter
	ConnAsyncCallback conn.AsyncCallback
}
