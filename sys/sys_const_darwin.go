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
)

const (
	EVFilterClosed = -0xd
	EVFilterWrite  = syscall.EVFILT_WRITE
	EVFilterRead   = syscall.EVFILT_READ
)
