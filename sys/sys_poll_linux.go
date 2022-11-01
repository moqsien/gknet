//go:build linux

package sys

import (
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/moqsien/processes/logger"

	"github.com/moqsien/gknet/utils"
)

var ePool = &sync.Pool{New: func() any {
	return &syscall.EpollEvent{}
}}

func eGet() *syscall.EpollEvent {
	return ePool.Get().(*syscall.EpollEvent)
}

func ePut(event *syscall.EpollEvent) {
	ePool.Put(event)
}

const (
	ReadEvents      = syscall.EPOLLPRI | syscall.EPOLLIN
	WriteEvents     = syscall.EPOLLOUT
	ReadWriteEvents = ReadEvents | WriteEvents
)

var eSysName map[int]string = map[int]string{
	syscall.EPOLL_CTL_ADD: "epoll_ctl_add",
	syscall.EPOLL_CTL_MOD: "epoll_ctl_mod",
	syscall.EPOLL_CTL_DEL: "epoll_ctl_del",
}

func epollFdHandler(pollFd, fd, ctlAction int, evs uint32) (err error) {
	var event *syscall.EpollEvent
	if ctlAction != syscall.EPOLL_CTL_DEL {
		event = eGet()
		defer ePut(event)
		event.Fd, event.Events = int32(fd), evs
	}
	err = syscall.EpollCtl(pollFd, ctlAction, fd, event)
	return utils.SysError(eSysName[ctlAction], err)
}

func AddReadWrite(pollFd, fd int) (err error) {
	return epollFdHandler(pollFd, fd, syscall.EPOLL_CTL_ADD, ReadWriteEvents)
}

func AddRead(pollFd, fd int) (err error) {
	return epollFdHandler(pollFd, fd, syscall.EPOLL_CTL_ADD, ReadEvents)
}

func AddWrite(pollFd, fd int) (err error) {
	return epollFdHandler(pollFd, fd, syscall.EPOLL_CTL_ADD, WriteEvents)
}

func ModRead(pollFd, fd int) (err error) {
	return epollFdHandler(pollFd, fd, syscall.EPOLL_CTL_MOD, ReadEvents)
}

func ModWrite(pollFd, fd int) (err error) {
	return epollFdHandler(pollFd, fd, syscall.EPOLL_CTL_MOD, WriteEvents)
}

func ModReadWrite(pollFd, fd int) (err error) {
	return epollFdHandler(pollFd, fd, syscall.EPOLL_CTL_MOD, ReadWriteEvents)
}

func UnRegister(pollFd, fd int) (err error) {
	return epollFdHandler(pollFd, fd, syscall.EPOLL_CTL_DEL, 0)
}

func expand(size int) (newSize int, events []syscall.EpollEvent) {
	newSize = size << 1
	events = make([]syscall.EpollEvent, newSize)
	return
}

func shrink(size int) (newSize int, events []syscall.EpollEvent) {
	newSize = size >> 1
	events = make([]syscall.EpollEvent, newSize)
	return
}

func WaitPoll(pollFd, pollEvFd int, w WaitCallback, doCallbackErr DoError) error {
	size := InitPollSize
	events := make([]syscall.EpollEvent, size)
	var (
		trigger      bool
		timeout      int = -1
		pollEvBuffer     = []byte{}
	)
	for {
		n, err := syscall.EpollWait(pollFd, events, timeout)
		if n == 0 || (n < 0 && err == syscall.EINTR) {
			timeout = -1
			runtime.Gosched()
			continue
		} else if err != nil {
			logger.Errorf("error occurs in epoll: %v", utils.SysError("epoll_wait", err))
			return err
		}
		timeout = 0
		for i := 0; i < n; i++ {
			ev := &events[i]
			fd := int(ev.Fd)
			if fd == pollEvFd {
				trigger = true
				syscall.Read(pollEvFd, pollEvBuffer)
			}
			if i == n-1 {
				trigger, err = w(fd, ev.Events, trigger)
			} else {
				trigger, err = w(fd, ev.Events, false)
			}
			err = doCallbackErr(err)
			if err != nil {
				return err
			}
		}

		if n == size && (size<<1 <= MaxPollSize) {
			size, events = expand(size)
		} else if (n < size>>1) && (size>>1 >= MinPollSize) {
			size, events = shrink(size)
		}
	}
}

func pEventFd(initval uint, flags int) (fd int, err error) {
	r0, _, e1 := syscall.Syscall(syscall.SYS_EVENTFD2, uintptr(initval), uintptr(flags), 0)
	fd, err = int(r0), errnoErr(e1)
	return
}

func CreatePoll() (pollFd, pollEvFd int, err error) {
	pollFd, err = syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		err = utils.SysError("epoll_create1", err)
		return
	}
	pollEvFd, err = pEventFd(0, syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC)
	if err != nil {
		syscall.Close(pollFd)
		err = utils.SysError("epoll_eventfd", err)
		return
	}
	err = AddRead(pollFd, pollEvFd)
	if err != nil {
		syscall.Close(pollFd)
		syscall.Close(pollEvFd)
		err = utils.SysError("epoll_eventfd_add", err)
		return
	}
	return
}

var (
	u uint64 = 1
	b        = (*(*[8]byte)(unsafe.Pointer(&u)))[:]
)

func Trigger(pollEvFd int) (err error) {
	if _, err = syscall.Write(pollEvFd, b); err == syscall.EAGAIN {
		err = nil
	}
	return utils.SysError("pollEvFd_write", err)
}

func Accept(listenerFd int, timeout ...time.Duration) (int, syscall.Sockaddr, error) {
	nfd, sock, err := syscall.Accept4(listenerFd, syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC)
	switch err {
	case nil:
		return nfd, sock, err
	default:
		return -1, nil, err
	case syscall.ENOSYS:
	case syscall.EINVAL:
	case syscall.EACCES:
	case syscall.EFAULT:
	}
	nfd, sock, err = syscall.Accept(listenerFd)
	if err == nil {
		syscall.CloseOnExec(nfd)
	} else {
		return -1, nil, err
	}
	if err = syscall.SetNonblock(nfd, true); err != nil {
		syscall.Close(nfd)
		return -1, nil, err
	}
	SetKeepAlive(nfd, timeout...)
	return nfd, sock, nil
}
