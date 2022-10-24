//go:build aix || freebsd || linux || netbsd

package sys

import "syscall"

const (
	TCP_KEEPINTVL = syscall.TCP_KEEPINTVL
	TCP_KEEPIDLE  = syscall.TCP_KEEPIDLE
	SOL_SOCKET    = syscall.SOL_SOCKET
	IPPROTO_TCP   = syscall.IPPROTO_TCP
	SO_KEEPALIVE  = syscall.SO_KEEPALIVE
)
