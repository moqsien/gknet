/*
sys just generalized some needed syscalls from different platforms.
*/
package sys

import (
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/moqsien/gknet/utils"
)

type EventHandler interface {
	WriteToFd() error
	AsyncWriteToFd()
	AsyncWriteToFdAndWait(wg *sync.WaitGroup)
	ReadFromFd() error
	AsyncReadFromFd()
	AsyncReadFromFdAndWait(wg *sync.WaitGroup)
	Close() error
	AsyncClose()
}

type WaitCallback func(fd int, events uint32, trigger bool, wg *sync.WaitGroup) (newTrigger bool, err error)

type DoError func(err error) error

const (
	MaxPollSize         = 1024
	MinPollSize         = 32
	InitPollSize        = 128
	EVFilterFd          = -0xd
	DefaultTCPKeepAlive = 15 // Seconds
)

func CloseFd(fd int) error {
	return utils.SysError("fd_close", syscall.Close(fd))
}

func transKeepAlive(t ...time.Duration) (secs int) {
	if len(t) == 0 {
		return DefaultTCPKeepAlive
	}
	secs = int(t[0] / time.Second)
	if secs >= 1 {
		return
	}
	return DefaultTCPKeepAlive
}

func SetKeepAlive(fd int, timeout ...time.Duration) (err error) {
	// timeout in seconds.
	secs := transKeepAlive(timeout...)
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

func SetReusePort(fd int) error {
	return utils.SysError("setsockopt", syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, SO_REUSEPORT, 1))
}

func SetReuseAddr(fd int) error {
	return utils.SysError("setsockopt", syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1))
}

func SetRecvBufferSize(fd, size int) error {
	return utils.SysError("setsockopt", syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF, size))
}

func SetSendBufferSize(fd, size int) error {
	return utils.SysError("setsockopt", syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_SNDBUF, size))
}

func HandleEvents(events uint32, handler EventHandler) (err error) {
	if events&ClosedFdEvents != 0 { // only for darwin.
		err = handler.Close()
		return
	}

	if events&OutEvents != 0 {
		err = handler.WriteToFd()
		if err != nil {
			return
		}
	}

	if events&InEvents != 0 {
		err = handler.ReadFromFd()
		if err != nil {
			return
		}
	}
	return
}

func AsyncHandleEvents(events uint32, handler EventHandler) {
	if events&ClosedFdEvents != 0 { // only for darwin.
		handler.AsyncClose()
		return
	}

	if events&OutEvents != 0 {
		handler.AsyncWriteToFd()
		return
	}

	if events&InEvents != 0 {
		handler.AsyncReadFromFd()
		return
	}
}

func AsyncHandleEventsAndWait(events uint32, handler EventHandler, wg *sync.WaitGroup) {
	if events&ClosedFdEvents != 0 { // only for darwin.
		handler.AsyncClose()
		return
	}

	if events&OutEvents != 0 {
		handler.AsyncWriteToFdAndWait(wg)
		return
	}

	if events&InEvents != 0 {
		handler.AsyncReadFromFdAndWait(wg)
		return
	}
}

var _zero uintptr

func bytes2iovec(bs [][]byte) []syscall.Iovec {
	iovecs := make([]syscall.Iovec, len(bs))
	for i, b := range bs {
		iovecs[i].SetLen(len(b))
		if len(b) > 0 {
			iovecs[i].Base = &b[0]
		} else {
			iovecs[i].Base = (*byte)(unsafe.Pointer(&_zero))
		}
	}
	return iovecs
}

func writev(fd int, iovs []syscall.Iovec) (n int, err error) {
	var _p0 unsafe.Pointer
	if len(iovs) > 0 {
		_p0 = unsafe.Pointer(&iovs[0])
	} else {
		_p0 = unsafe.Pointer(&_zero)
	}
	r0, _, e1 := syscall.Syscall(syscall.SYS_WRITEV, uintptr(fd), uintptr(_p0), uintptr(len(iovs)))
	n = int(r0)
	err = errnoErr(e1)
	return
}

func readv(fd int, iovs []syscall.Iovec) (n int, err error) {
	var _p0 unsafe.Pointer
	if len(iovs) > 0 {
		_p0 = unsafe.Pointer(&iovs[0])
	} else {
		_p0 = unsafe.Pointer(&_zero)
	}
	r0, _, e1 := syscall.Syscall(syscall.SYS_READV, uintptr(fd), uintptr(_p0), uintptr(len(iovs)))
	n = int(r0)
	err = errnoErr(e1)
	return
}

func Writev(fd int, iovs [][]byte) (n int, err error) {
	iovecs := bytes2iovec(iovs)
	n, err = writev(fd, iovecs)
	return n, err
}

func Readv(fd int, iovs [][]byte) (n int, err error) {
	iovecs := bytes2iovec(iovs)
	n, err = readv(fd, iovecs)
	return n, err
}

func Write(fd int, p []byte) (n int, err error) {
	return syscall.Write(fd, p)
}

func Read(fd int, p []byte) (n int, err error) {
	if fd == 0 {
		return 0, nil
	}
	return syscall.Read(fd, p)
}

func WriteUdp(fd int, p []byte, flags int, to syscall.Sockaddr) (err error) {
	return syscall.Sendto(fd, p, flags, to)
}
