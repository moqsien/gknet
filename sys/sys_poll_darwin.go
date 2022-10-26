//go:build amd64 && darwin

package sys

import (
	"errors"
	"runtime"
	"sync"
	"syscall"

	"github.com/moqsien/processes/logger"

	"github.com/moqsien/gknet/utils"
)

const (
	kSysAdd = "kevent_add"
	kSysMod = "kevent_del"
	kSysDel = "kevent_del"
)

var kFilters sync.Map = sync.Map{}

func getKevents(oldEvents, newEvents uint32, fd int) (r []syscall.Kevent_t) {
	if newEvents&InEvents != 0 && oldEvents&InEvents == 0 {
		r = append(r, syscall.Kevent_t{
			Ident:  uint64(fd),
			Flags:  syscall.EV_ADD | syscall.EV_ENABLE,
			Filter: syscall.EVFILT_READ,
		})
	}

	if newEvents&InEvents == 0 && oldEvents&InEvents != 0 {
		r = append(r, syscall.Kevent_t{
			Ident:  uint64(fd),
			Flags:  syscall.EV_DELETE | syscall.EV_ONESHOT,
			Filter: syscall.EVFILT_READ,
		})
	}

	if newEvents&OutEvents != 0 && oldEvents&OutEvents == 0 {
		r = append(r, syscall.Kevent_t{
			Ident:  uint64(fd),
			Flags:  syscall.EV_ADD | syscall.EV_ENABLE,
			Filter: syscall.EVFILT_WRITE,
		})
	}

	if newEvents&OutEvents == 0 && oldEvents&OutEvents != 0 {
		r = append(r, syscall.Kevent_t{
			Ident:  uint64(fd),
			Flags:  syscall.EV_DELETE | syscall.EV_ONESHOT,
			Filter: syscall.EVFILT_WRITE,
		})
	}
	return
}

func AddReadWrite(pollFd, fd int) (err error) {
	kFilters.Store(fd, InAndOutEvents)
	kevents := getKevents(NoneEvents, InAndOutEvents, fd)
	_, err = syscall.Kevent(pollFd, kevents, nil, nil)
	return utils.SysError(kSysAdd, err)
}

func AddRead(pollFd, fd int) (err error) {
	kFilters.Store(fd, InEvents)
	kevents := getKevents(NoneEvents, InEvents, fd)
	_, err = syscall.Kevent(pollFd, kevents, nil, nil)
	return utils.SysError(kSysAdd, err)
}

func AddWrite(pollFd, fd int) (err error) {
	kFilters.Store(fd, OutEvents)
	kevents := getKevents(NoneEvents, OutEvents, fd)
	_, err = syscall.Kevent(pollFd, kevents, nil, nil)
	return utils.SysError(kSysAdd, err)
}

func ModRead(pollFd, fd int) (err error) {
	oldEvents, ok := kFilters.Load(fd)
	if !ok {
		return utils.SysError(kSysMod, errors.New("fd not added!"))
	}
	kevents := getKevents(oldEvents.(uint32), InEvents, fd)
	_, err = syscall.Kevent(pollFd, kevents, nil, nil)
	return utils.SysError(kSysDel, err)
}

func ModWrite(pollFd, fd int) (err error) {
	oldEvents, ok := kFilters.Load(fd)
	if !ok {
		return utils.SysError(kSysMod, errors.New("fd not added!"))
	}
	kevents := getKevents(oldEvents.(uint32), OutEvents, fd)
	_, err = syscall.Kevent(pollFd, kevents, nil, nil)
	return utils.SysError(kSysMod, err)
}

func ModReadWrite(pollFd, fd int) (err error) {
	oldEvents, ok := kFilters.Load(fd)
	if !ok {
		return utils.SysError(kSysMod, errors.New("fd not added!"))
	}
	kevents := getKevents(oldEvents.(uint32), InAndOutEvents, fd)
	_, err = syscall.Kevent(pollFd, kevents, nil, nil)
	return utils.SysError(kSysMod, err)
}

func UnRegister(pollFd, fd int) (err error) {
	oldEvents, ok := kFilters.Load(fd)
	if !ok {
		return utils.SysError(kSysDel, errors.New("fd not added!"))
	}
	kevents := getKevents(oldEvents.(uint32), NoneEvents, fd)
	_, err = syscall.Kevent(pollFd, kevents, nil, nil)
	if err == nil {
		kFilters.Delete(fd)
	}
	return utils.SysError(kSysDel, err)
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

func WaitPoll(pollFd, _pollEvFd int, w WaitCallback, doCallbackErr DoError) error {
	size := InitPollSize
	eventList := make([]syscall.Kevent_t, size)
	var (
		ts      syscall.Timespec
		tsp     *syscall.Timespec
		trigger bool
	)

	for {
		n, err := syscall.Kevent(pollFd, nil, eventList, tsp)
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
			ev := eventList[i]
			fd := int(ev.Ident)
			if fd == 0 {
				trigger = true
			}
			evFilter = ev.Filter
			var events uint32
			if (ev.Flags&syscall.EV_EOF != 0) || (ev.Flags&syscall.EV_ERROR != 0) {
				events |= ClosedFdEvents
			}
			if evFilter == syscall.EVFILT_WRITE && ev.Flags&syscall.EV_ENABLE != 0 {
				events |= OutEvents
			}
			if evFilter == syscall.EVFILT_READ && ev.Flags&syscall.EV_ENABLE != 0 {
				events |= InEvents
			}
			if i != n-1 {
				trigger, err = w(fd, events, false)
			} else {
				trigger, err = w(fd, events, trigger)
			}
			err = doCallbackErr(err)
			if err != nil {
				return err
			}
		}

		if n == size && (size<<1 <= MaxPollSize) {
			size, eventList = expand(size)
		} else if (n < size>>1) && (size>>1 >= MinPollSize) {
			size, eventList = shrink(size)
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
