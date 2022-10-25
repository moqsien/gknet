//go:build amd64 && darwin

package sys

import (
	"runtime"
	"syscall"

	"github.com/moqsien/processes/logger"

	"github.com/moqsien/gknet/utils"
)

const (
	kSysAdd = "kevent_add"
	kSysDel = "kevent_del"
)

func AddReadWrite(pollFd, fd int) (err error) {
	_, err = syscall.Kevent(pollFd,
		[]syscall.Kevent_t{
			{Ident: uint64(fd), Flags: syscall.EV_ADD, Filter: syscall.EVFILT_READ},
			{Ident: uint64(fd), Flags: syscall.EV_ADD, Filter: syscall.EVFILT_WRITE},
		},
		nil,
		nil)
	return utils.SysError(kSysAdd, err)
}

func AddRead(pollFd, fd int) (err error) {
	_, err = syscall.Kevent(pollFd, []syscall.Kevent_t{
		{Ident: uint64(fd), Flags: syscall.EV_ADD, Filter: syscall.EVFILT_READ},
	}, nil, nil)
	return utils.SysError(kSysAdd, err)
}

func AddWrite(pollFd, fd int) (err error) {
	_, err = syscall.Kevent(pollFd, []syscall.Kevent_t{
		{Ident: uint64(fd), Flags: syscall.EV_ADD, Filter: syscall.EVFILT_WRITE},
	}, nil, nil)
	return utils.SysError(kSysAdd, err)
}

func ModRead(pollFd, fd int) (err error) {
	_, err = syscall.Kevent(pollFd, []syscall.Kevent_t{
		{Ident: uint64(fd), Flags: syscall.EV_DELETE, Filter: syscall.EVFILT_WRITE},
	}, nil, nil)
	return utils.SysError(kSysDel, err)
}

func ModWrite(pollFd, fd int) (err error) {
	return ModReadWrite(pollFd, fd)
}

func ModReadWrite(pollFd, fd int) (err error) {
	_, err = syscall.Kevent(pollFd, []syscall.Kevent_t{
		{Ident: uint64(fd), Flags: syscall.EV_ADD, Filter: syscall.EVFILT_WRITE},
	}, nil, nil)
	return utils.SysError(kSysAdd, err)
}

func UnRegister(pollFd, fd int) (err error) {
	return nil
}

func expand(size int) (newSize int, events []syscall.Kevent_t) {
	newSize = size << 1
	events = make([]syscall.Kevent_t, newSize)
	return
}

func shrink(size int) (newSize int, events []syscall.Kevent_t) {
	newSize = size >> 1
	events = make([]syscall.Kevent_t, newSize)
	return
}

func WaitPoll(pollFd, pollEvFd int, w WaitCallback, doCallbackErr DoError) error {
	size := InitPollSize
	events := make([]syscall.Kevent_t, size)
	var (
		ts      syscall.Timespec
		tsp     *syscall.Timespec
		trigger bool
	)

	for {
		n, err := syscall.Kevent(pollEvFd, nil, events, tsp)
		if n == 0 || (n < 0 && err == syscall.EINTR) {
			tsp = nil
			runtime.Gosched()
			continue
		} else if err != nil {
			logger.Errorf("error occurs in kqueue: %v", utils.SysError("kevent_wait", err))
			return err
		}
		tsp = &ts

		var evFilter int16
		for i := 0; i < n; i++ {
			ev := events[i]
			fd := int(ev.Ident)
			if fd == pollEvFd {
				trigger = true
			}
			evFilter = ev.Filter
			if (ev.Flags&syscall.EV_EOF != 0) || (ev.Flags&syscall.EV_ERROR != 0) {
				evFilter = EVFilterClosed
			}
			if i != n-1 {
				trigger, err = w(fd, int64(evFilter), false)
			} else {
				trigger, err = w(fd, int64(evFilter), trigger)
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

func CreatePoll() (pollFd, pollEvFd int, err error) {
	pollFd, err = syscall.Kqueue()
	if err != nil {
		err = utils.SysError("kqueue", err)
		return
	}

	_, err = syscall.Kevent(pollFd, []syscall.Kevent_t{{
		Ident:  0,
		Filter: syscall.EVFILT_USER,
		Flags:  syscall.EV_ADD | syscall.EV_CLEAR,
	}}, nil, nil)
	if err != nil {
		syscall.Close(pollFd)
		err = utils.SysError("kqueue_eventfd", err)
		return
	}
	pollEvFd = pollFd
	return
}

var noteTrigger = []syscall.Kevent_t{{
	Ident:  0,
	Filter: syscall.EVFILT_USER,
	Fflags: syscall.NOTE_TRIGGER,
}}

func Trigger(pollFd int) (err error) {
	if _, err = syscall.Kevent(pollFd, noteTrigger, nil, nil); err == syscall.EAGAIN {
		err = nil
	}
	return utils.SysError("kevent_trigger", err)
}

func Accept(listenerFd int, timeout ...int) (int, syscall.Sockaddr, error) {
	nfd, sa, err := syscall.Accept(listenerFd)
	if err == nil {
		syscall.CloseOnExec(nfd)
	}
	if err != nil {
		return -1, nil, err
	}
	if err = syscall.SetNonblock(nfd, true); err != nil {
		syscall.Close(nfd)
		return -1, nil, err
	}
	SetKeepAlive(nfd, timeout...)
	return nfd, sa, nil
}

func HandleEvents(events int64, handler EventHandler) (err error) {
	switch events {
	case EVFilterClosed:
		err = handler.Close(syscall.ECONNRESET)
	case EVFilterWrite:
		err = handler.WriteToFd()
	case EVFilterRead:
		err = handler.ReadFromFd()
	}
	return
}
