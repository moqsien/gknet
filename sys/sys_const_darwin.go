//go:build amd64 && darwin

package sys

import "syscall"

const (
	TCP_KEEPINTVL = 0x101
	TCP_KEEPALIVE = syscall.TCP_KEEPALIVE
	TCP_KEEPIDLE  = TCP_KEEPALIVE
	SOL_SOCKET    = syscall.SOL_SOCKET
	IPPROTO_TCP   = syscall.IPPROTO_TCP
	SO_KEEPALIVE  = syscall.SO_KEEPALIVE
	SO_REUSEPORT  = 0x200
)

const (
	EVFilterClosed = -0xd
	EVFilterWrite  = syscall.EVFILT_WRITE
	EVFilterRead   = syscall.EVFILT_READ
)

const (
	InEvents       uint32 = 0x2
	OutEvents      uint32 = 0x4
	ClosedFdEvents uint32 = 0x8
	InAndOutEvents uint32 = InEvents | OutEvents
	NoneEvents     uint32 = 0
)

const (
	EAGAIN     = syscall.EAGAIN
	ECONNRESET = syscall.ECONNRESET
)
