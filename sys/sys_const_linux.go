//go:build aix || freebsd || linux || netbsd

package sys

import "syscall"

const (
	TCP_KEEPINTVL = syscall.TCP_KEEPINTVL
	TCP_KEEPIDLE  = syscall.TCP_KEEPIDLE
	SOL_SOCKET    = syscall.SOL_SOCKET
	IPPROTO_TCP   = syscall.IPPROTO_TCP
	SO_KEEPALIVE  = syscall.SO_KEEPALIVE
	SO_REUSEPORT  = 0xf
)

const (
	ErrEvents      = syscall.EPOLLERR | syscall.EPOLLHUP | syscall.EPOLLRDHUP
	OutEvents      = ErrEvents | syscall.EPOLLOUT
	InEvents       = ErrEvents | syscall.EPOLLIN | syscall.EPOLLPRI
	ClosedFdEvents = 0
)

const (
	EAGAIN     = syscall.EAGAIN
	ECONNRESET = syscall.ECONNRESET
	EINVAL     = syscall.EINVAL
	ENOENT     = syscall.ENOENT
)
