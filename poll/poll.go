package poll

import (
	"sync/atomic"

	"github.com/moqsien/processes/logger"

	"github.com/moqsien/gknet/sys"
	"github.com/moqsien/gknet/utils"
	"github.com/moqsien/gknet/utils/errs"
	"github.com/moqsien/gknet/utils/queue"
)

const (
	MaxSize  = 1024
	MinSize  = 32
	IniSize  = 128
	MaxTasks = 256
)

type PollerCallBack func(fd int, events uint32) error

type Poller struct {
	pollFd     int             // poll file descriptor
	pollEvFd   int             // poll event file descriptor
	priorTasks queue.TaskQueue // tasks with priority
	tasks      queue.TaskQueue // tasks
	toTrigger  int32           // atomic number to trigger tasks
	Eloop      IELoop          // eventloop
	Buffer     []byte          // buffer for reading from fd
}

func New() (p *Poller, err error) {
	p = new(Poller)
	p.pollFd, p.pollEvFd, err = sys.CreatePoll()
	if err != nil {
		p = nil
		return
	}
	p.priorTasks = queue.NewQueue()
	p.tasks = queue.NewQueue()
	return
}

func (that *Poller) AddTask(f PollTaskFunc, arg PollTaskArg) (err error) {
	task := GetTask()
	task.Go, task.Arg = f, arg
	that.tasks.Enqueue(task)
	if atomic.CompareAndSwapInt32(&that.toTrigger, 0, 1) {
		err = sys.Trigger(that.pollEvFd)
	}
	return
}

func (that *Poller) AddPriorTask(f PollTaskFunc, arg PollTaskArg) (err error) {
	task := GetTask()
	task.Go, task.Arg = f, arg
	that.priorTasks.Enqueue(task)
	if atomic.CompareAndSwapInt32(&that.toTrigger, 0, 1) {
		err = sys.Trigger(that.pollEvFd)
	}
	return
}

func (that *Poller) Start(fn PollerCallBack) error {
	var wcb sys.WaitCallback = func(fd int, events uint32, trigger bool) (bool, error) {
		var err error
		err = fn(fd, events)
		if err != nil {
			return false, err
		}

		if trigger {
			trigger = false
			t := that.priorTasks.Dequeue()
			for ; t != nil; t = that.priorTasks.Dequeue() {
				task := t.(*PollTask)
				switch err = task.Go(task.Arg); err {
				case nil:
				case errs.ErrEngineShutdown, errs.ErrAcceptSocket:
					return trigger, err
				default:
					logger.Warningf("error occurs in user-defined function, %v", err)
				}
				PutTask(task)
			}
			for i := 0; i < MaxTasks; i++ {
				if t = that.tasks.Dequeue(); t == nil {
					break
				}
				task := t.(*PollTask)
				switch err = task.Go(task.Arg); err {
				case nil:
				case errs.ErrEngineShutdown, errs.ErrAcceptSocket:
					return trigger, err
				default:
					logger.Warningf("error occurs in user-defined function, %v", err)
				}
				PutTask(task)
			}
			atomic.StoreInt32(&that.toTrigger, 0)
			if (!that.tasks.IsEmpty() || !that.priorTasks.IsEmpty()) && atomic.CompareAndSwapInt32(&that.toTrigger, 0, 1) {
				if err := sys.Trigger(that.pollEvFd); err == nil {
					trigger = true
				}
			}
		}
		return trigger, err
	}
	return sys.WaitPoll(that.pollFd, that.pollEvFd, wcb, doWaitCallbackErr)
}

func doWaitCallbackErr(err error) error {
	switch err {
	case nil:
		return nil
	case errs.ErrAcceptSocket, errs.ErrEngineShutdown:
		return err
	default:
		logger.Warningf("Error occurs in eventloop: %v", err)
		return nil
	}
}

func (that *Poller) Close() error {
	if err := utils.SysError("pollfd_close", sys.CloseFd(that.pollFd)); err != nil {
		return err
	}
	if that.pollFd != that.pollEvFd {
		return utils.SysError("pollEvFd_close", sys.CloseFd(that.pollEvFd))
	}
	return nil
}

func (that *Poller) AddReadWrite(fd IFd) error {
	return sys.AddReadWrite(that.pollFd, fd.GetFd())
}

func (that *Poller) AddRead(fd IFd) error {
	return sys.AddRead(that.pollFd, fd.GetFd())
}

func (that *Poller) AddWrite(fd IFd) error {
	return sys.AddWrite(that.pollFd, fd.GetFd())
}

func (that *Poller) ModReadWrite(fd IFd) error {
	return sys.ModReadWrite(that.pollFd, fd.GetFd())
}

func (that *Poller) ModRead(fd IFd) error {
	return sys.ModRead(that.pollFd, fd.GetFd())
}

func (that *Poller) ModWrite(fd IFd) error {
	return sys.ModWrite(that.pollFd, fd.GetFd())
}

func (that *Poller) RemoveFd(fd IFd) error {
	return sys.UnRegister(that.pollFd, fd.GetFd())
}
