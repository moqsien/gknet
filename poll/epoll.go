package poll

import (
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/moqsien/processes/logger"
	"golang.org/x/sys/unix"

	"github.com/moqsien/gknet/utils/errs"
	"github.com/moqsien/gknet/utils/queue"
)

const (
	ReadEvents      = unix.EPOLLPRI | unix.EPOLLIN
	WriteEvents     = unix.EPOLLOUT
	ReadWriteEvents = ReadEvents | WriteEvents
	DefaultTimeout  = -1
	MaxSize         = 1024
	MinSize         = 32
	IniSize         = 128
	MaxTasks        = 256
)

var (
	u uint64 = 1
	b        = (*(*[8]byte)(unsafe.Pointer(&u)))[:]
)

type IFd interface {
	GetFd() int
}

type PollerCallBack func(fd int, events uint32) error

type Poller struct {
	PollFd          int
	PollEventFd     int
	PollEventBuffer []byte
	TaskQueue       queue.TaskQueue
	PriorTaskQueue  queue.TaskQueue
	Eloop           IEPool
	pool            *sync.Pool
	timeout         int
	size            int
	toWakeup        int32
	toDotask        bool
	eventList       []unix.EpollEvent
}

func NewPoller() (p *Poller) {
	p = new(Poller)
	var err error
	if p.PollFd, err = unix.EpollCreate1(unix.EPOLL_CLOEXEC); err != nil {
		p = nil
		err = os.NewSyscallError("epoll_create1", err)
		return
	}
	if p.PollEventFd, err = unix.Eventfd(0, unix.EFD_NONBLOCK|unix.EFD_CLOEXEC); err != nil {
		_ = p.Close()
		p = nil
		err = os.NewSyscallError("eventfd", err)
		return
	}
	p.PollEventBuffer = make([]byte, 8)
	if err = p.AddRead(p); err != nil {
		_ = p.Close()
		p = nil
		return
	}
	p.TaskQueue = queue.NewQueue()
	p.PriorTaskQueue = queue.NewQueue()
	p.pool = &sync.Pool{New: func() interface{} { return unix.EpollEvent{} }}
	p.timeout = DefaultTimeout
	p.size = IniSize
	p.eventList = make([]unix.EpollEvent, p.size)
	return
}

func (that *Poller) ExpandEventList() {
	if newSize := that.size << 1; newSize <= MaxSize {
		that.size = newSize
		that.eventList = make([]unix.EpollEvent, newSize)
	}
}

func (that *Poller) ShrinkEventList() {
	if newSize := that.size >> 1; newSize >= MinSize {
		that.size = newSize
		that.eventList = make([]unix.EpollEvent, newSize)
	}
}

func (that *Poller) AddTask(f PollTaskFunc, arg PollTaskArg) (err error) {
	task := GetTask()
	task.Go, task.Arg = f, arg
	that.TaskQueue.Enqueue(task)
	if atomic.CompareAndSwapInt32(&that.toWakeup, 0, 1) {
		if _, err = unix.Write(that.PollEventFd, b); err == unix.EAGAIN {
			err = nil
		}
	}
	return os.NewSyscallError("write", err)
}

func (that *Poller) AddPriorTask(f PollTaskFunc, arg PollTaskArg) (err error) {
	task := GetTask()
	task.Go, task.Arg = f, arg
	that.PriorTaskQueue.Enqueue(task)
	if atomic.CompareAndSwapInt32(&that.toWakeup, 0, 1) {
		if _, err = unix.Write(that.PollEventFd, b); err == unix.EAGAIN {
			err = nil
		}
	}
	return os.NewSyscallError("write", err)
}

func (that *Poller) Start(fn PollerCallBack) error {
	for {
		n, err := unix.EpollWait(that.PollFd, that.eventList, that.timeout)
		if n == 0 || (n < 0 && err == unix.EINTR) {
			that.timeout = -1
			runtime.Gosched()
			continue
		} else if err != nil {
			logger.Errorf("error occurs in epoll: %v", os.NewSyscallError("epoll_wait", err))
			return err
		}
		that.timeout = 0

		for i := 0; i < n; i++ {
			ev := &that.eventList[i]
			if fd := int(ev.Fd); fd != that.PollEventFd {
				switch err = fn(fd, ev.Events); err {
				case nil:
				case errs.ErrAcceptSocket, errs.ErrEngineShutdown:
					return err
				default:
					logger.Warningf("error occurs in event-loop: %v", err)
				}
			} else {
				that.toDotask = true
				_, _ = unix.Read(that.PollEventFd, that.PollEventBuffer)
			}
		}

		if that.toDotask {
			that.toDotask = false
			t := that.PriorTaskQueue.Dequeue()
			for ; t != nil; t = that.PriorTaskQueue.Dequeue() {
				task := t.(*PollTask)
				switch err = task.Go(task.Arg); err {
				case nil:
				case errs.ErrEngineShutdown:
					return err
				default:
					logger.Warningf("error occurs in user-defined function, %v", err)
				}
				PutTask(task)
			}
			for i := 0; i < MaxTasks; i++ {
				if t = that.TaskQueue.Dequeue(); t == nil {
					break
				}
				task := t.(*PollTask)
				switch err = task.Go(task.Arg); err {
				case nil:
				case errs.ErrEngineShutdown:
					return err
				default:
					logger.Warningf("error occurs in user-defined function, %v", err)
				}
				PutTask(task)
			}
			atomic.StoreInt32(&that.toWakeup, 0)
			if (!that.TaskQueue.IsEmpty() || !that.PriorTaskQueue.IsEmpty()) && atomic.CompareAndSwapInt32(&that.toWakeup, 0, 1) {
				switch _, err = unix.Write(that.PollEventFd, b); err {
				case nil, unix.EAGAIN:
				default:
					that.toDotask = true
				}
			}
		}

		if n == that.size {
			that.ExpandEventList()
		} else if n < that.size>>1 {
			that.ShrinkEventList()
		}
	}
}

func (that *Poller) GetFd() int {
	return that.PollEventFd
}

func (that *Poller) Close() error {
	if err := os.NewSyscallError("close", unix.Close(that.PollFd)); err != nil {
		return err
	}
	return os.NewSyscallError("close", unix.Close(that.PollEventFd))
}

/* add events to epoll */
func (that *Poller) AddReadWrite(fd IFd) (err error) {
	event := that.pool.Get().(unix.EpollEvent)
	event.Fd, event.Events = int32(fd.GetFd()), ReadWriteEvents
	err = os.NewSyscallError("epoll_ctl_add", unix.EpollCtl(that.PollFd, unix.EPOLL_CTL_ADD, fd.GetFd(), &event))
	that.pool.Put(event)
	if err != nil {
		return
	}
	return
}

func (that *Poller) AddRead(fd IFd) (err error) {
	event := that.pool.Get().(unix.EpollEvent)
	event.Fd, event.Events = int32(fd.GetFd()), ReadEvents
	err = os.NewSyscallError("epoll_ctl_add", unix.EpollCtl(that.PollFd, unix.EPOLL_CTL_ADD, fd.GetFd(), &event))
	that.pool.Put(event)
	if err != nil {
		return
	}
	return
}

func (that *Poller) AddWrite(fd IFd) (err error) {
	event := that.pool.Get().(unix.EpollEvent)
	event.Fd, event.Events = int32(fd.GetFd()), WriteEvents
	err = os.NewSyscallError("epoll_ctl_add", unix.EpollCtl(that.PollFd, unix.EPOLL_CTL_ADD, fd.GetFd(), &event))
	that.pool.Put(event)
	if err != nil {
		return
	}
	return
}

func (that *Poller) ModRead(fd IFd) (err error) {
	event := that.pool.Get().(unix.EpollEvent)
	event.Fd, event.Events = int32(fd.GetFd()), ReadEvents
	err = os.NewSyscallError("epoll_ctl_mod", unix.EpollCtl(that.PollFd, unix.EPOLL_CTL_MOD, fd.GetFd(), &event))
	that.pool.Put(event)
	return
}

func (that *Poller) ModWrite(fd IFd) (err error) {
	event := that.pool.Get().(unix.EpollEvent)
	event.Fd, event.Events = int32(fd.GetFd()), WriteEvents
	err = os.NewSyscallError("epoll_ctl_mod", unix.EpollCtl(that.PollFd, unix.EPOLL_CTL_MOD, fd.GetFd(), &event))
	that.pool.Put(event)
	return
}

func (that *Poller) ModReadWrite(fd IFd) (err error) {
	event := that.pool.Get().(unix.EpollEvent)
	event.Fd, event.Events = int32(fd.GetFd()), ReadWriteEvents
	err = os.NewSyscallError("epoll_ctl_mod", unix.EpollCtl(that.PollFd, unix.EPOLL_CTL_MOD, fd.GetFd(), &event))
	that.pool.Put(event)
	return
}

func (that *Poller) RemoveFd(fd IFd) (err error) {
	err = os.NewSyscallError("epoll_ctl_del",
		unix.EpollCtl(that.PollFd, unix.EPOLL_CTL_DEL, fd.GetFd(), nil))
	return
}
