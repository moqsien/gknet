package sys

import (
	"runtime"
	"syscall"

	"github.com/moqsien/gknet/utils"
)

type EventHandler interface {
	WriteToFd() error
	ReadFromFd() error
	Close() error
}

type WaitCallback func(fd int, events int64, trigger bool) (newTrigger bool, err error)

type DoError func(err error) error

const (
	MaxPollSize         = 1024
	MinPollSize         = 32
	InitPollSize        = 128
	EVFilterFd          = -0xd
	DefaultTCPKeepAlive = 15 // Seconds
)

func CloseFd(fd int) error {
	return syscall.Close(fd)
}

func SetKeepAlive(fd int, timeout ...int) (err error) {
	// timeout in seconds.
	secs := DefaultTCPKeepAlive
	if len(timeout) > 0 && timeout[0] > 0 {
		secs = timeout[0]
	}
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, SO_KEEPALIVE, 1)
	if err != nil {
		return utils.SysError("setsockopt", err)
	}
	err = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_KEEPINTVL, secs)
	if err != nil {
		return utils.SysError("setsockopt", err)
	}
	err = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_KEEPIDLE, secs)
	runtime.KeepAlive(fd)
	return utils.SysError("setsockopt", err)
}
