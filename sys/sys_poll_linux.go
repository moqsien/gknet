//go:build linux

package sys

import (
	"os"
	"runtime"
	"sync"
	"syscall"

	"github.com/moqsien/processes/logger"

	"github.com/moqsien/gknet/utils"
	"github.com/moqsien/gknet/utils/errs"
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

func epollFdHandler(pollFd, fd, ctlAction int, evs uint32) (err error) {
	var event *syscall.EpollEvent
	if ctlAction != syscall.EPOLL_CTL_DEL {
		event = eGet()
		defer ePut(event)
		event.Fd, event.Events = int32(fd), evs
	}
	err = syscall.EpollCtl(pollFd, ctlAction, fd, event)
	var eSysName string
	switch ctlAction {
	case syscall.EPOLL_CTL_ADD:
		eSysName = "epoll_ctl_add"
	case syscall.EPOLL_CTL_MOD:
		eSysName = "epoll_ctl_mod"
	case syscall.EPOLL_CTL_DEL:
		eSysName = "epoll_ctl_del"
	default:
	}
	return utils.SysError(eSysName, err)
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

func Wait(pollFd, pollEvFd int, w WaitCallback) error {
	events := make([]syscall.EpollEvent, InitPollSize)
	var (
		trigger bool
		timeout int = -1
	)
	for {
		n, err := syscall.EpollWait(pollFd, events, timeout)
		err = utils.SysError("epoll_wait", err)
		if n == 0 || (n < 0 && err == syscall.EINTR) {
			timeout = -1
			runtime.Gosched()
			continue
		} else if err != nil {
			logger.Errorf("error occurs in epoll: %v", os.NewSyscallError("epoll_wait", err))
			return err
		}
		timeout = 0
		for i := 0; i < n; i++ {
			ev := &events[i]
			fd := int(ev.Fd)
			if fd == pollEvFd {
				trigger = true
			}
			if i == n-1 {
				err = w(fd, int64(ev.Events), trigger)
			} else {
				err = w(fd, int64(ev.Events), false)
			}
			switch err {
			case nil:
			case errs.ErrAcceptSocket, errs.ErrEngineShutdown:
				return err
			default:
				logger.Warningf("Error occurs in eventloop: %v", err)
			}
		}
		trigger = false
	}
}
