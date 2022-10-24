//go:build amd64 && darwin

package sys

import (
	"syscall"

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

func ModReadWrite(pollFd, fd int) (err error) {
	_, err = syscall.Kevent(pollFd, []syscall.Kevent_t{
		{Ident: uint64(fd), Flags: syscall.EV_ADD, Filter: syscall.EVFILT_WRITE},
	}, nil, nil)
	return utils.SysError("kevent add", err)
}

func UnRegister(pollFd, fd int) (err error) {
	return nil
}

var kevs []syscall.Kevent_t

func PollWait(pollFd, timeout int, tsp *syscall.Timespec, r *GkEventList) (n int, err error) {
	n, err = syscall.Kevent(pollFd, nil, kevs, tsp)
	for i := 0; i < n; i++ {
		ev := GkGet()
		ev.Fd = kevs[i].Ident
		ev.Event = int64(kevs[i].Filter)
		*r = append(*r, ev)
	}
	return
}
