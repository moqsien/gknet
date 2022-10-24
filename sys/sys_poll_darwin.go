//go:build amd64 && darwin

package sys

import (
	"runtime"
	"syscall"

	"github.com/moqsien/processes/logger"

	"github.com/moqsien/gknet/utils"
	"github.com/moqsien/gknet/utils/errs"
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

func ModReadWrite(pollFd, fd int) (err error) {
	_, err = syscall.Kevent(pollFd, []syscall.Kevent_t{
		{Ident: uint64(fd), Flags: syscall.EV_ADD, Filter: syscall.EVFILT_WRITE},
	}, nil, nil)
	return utils.SysError("kevent add", err)
}

func UnRegister(pollFd, fd int) (err error) {
	return nil
}

func WaitPoll(pollFd, pollEvFd int, w WaitCallback) error {
	events := make([]syscall.Kevent_t, InitPollSize)
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
				evFilter = EVFilterFd
			}
			if i != n-1 {
				err = w(fd, int64(evFilter), false)
			} else {
				err = w(fd, int64(evFilter), trigger)
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
	pollEvFd = 0
	return
}
