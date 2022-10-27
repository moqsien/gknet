package sys

import (
	"runtime"
	"syscall"
	"unsafe"

	"github.com/moqsien/gknet/utils"
)

type EventHandler interface {
	WriteToFd() error
	ReadFromFd() error
	Close(err ...syscall.Errno) error
}

type WaitCallback func(fd int, events uint32, trigger bool) (newTrigger bool, err error)

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

func HandleEvents(events uint32, handler EventHandler) (err error) {
	if events&ClosedFdEvents != 0 {
		err = handler.Close(syscall.ECONNRESET)
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
	err = e1
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
	err = e1
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
	return syscall.Read(fd, p)
}
