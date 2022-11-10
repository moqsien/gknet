/*
Poller provides an encapsulation of methods provided by package sys which is a generalization of syscalls from different platforms.
*/
package poll

import (
	"sync"
	"sync/atomic"

	"github.com/moqsien/processes/logger"
	"github.com/panjf2000/ants/v2"

	"github.com/moqsien/gknet/iface"
	"github.com/moqsien/gknet/sys"
	"github.com/moqsien/gknet/utils"
	"github.com/moqsien/gknet/utils/errs"
	"github.com/moqsien/gknet/utils/queue"
)

type Poller struct {
	pollFd      int             // poll file descriptor
	pollEvFd    int             // poll event file descriptor
	priorTasks  queue.TaskQueue // tasks with priority
	tasks       queue.TaskQueue // tasks
	toTrigger   int32           // atomic number to trigger tasks
	Eloop       iface.IELoop    // eventloop
	Buffer      []byte          // buffer for reading from fd
	Pool        *ants.Pool      // goroutine pool for running tasks
	ErrInfoChan chan error      // channel for sending error info
	wg          *sync.WaitGroup // wait for tasks to complete
}

func (that *Poller) GetFd() int {
	return that.pollFd
}

func (that *Poller) GetPollEvFd() int {
	return that.pollEvFd
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
	p.wg = &sync.WaitGroup{}
	return
}

func (that *Poller) AddTask(f iface.PollTaskFunc, arg iface.PollTaskArg) (err error) {
	task := GetTask()
	task.Go, task.Arg = f, arg
	that.tasks.Enqueue(task)
	if atomic.CompareAndSwapInt32(&that.toTrigger, 0, 1) {
		err = sys.Trigger(that.pollEvFd)
	}
	return
}

func (that *Poller) AddPriorTask(f iface.PollTaskFunc, arg iface.PollTaskArg) (err error) {
	task := GetTask()
	task.Go, task.Arg = f, arg
	that.priorTasks.Enqueue(task)
	if atomic.CompareAndSwapInt32(&that.toTrigger, 0, 1) {
		err = sys.Trigger(that.pollEvFd)
	}
	return
}

func (that *Poller) runTask(task *PollTask) (err error) {
	if that.Pool == nil {
		switch err = task.Go(task.Arg); err {
		case nil:
		case errs.ErrEngineShutdown, errs.ErrAcceptSocket:
			that.wg.Done()
			return err
		default:
			logger.Warningf("error occurs in user-defined function, %v", err)
			that.wg.Done()
			return nil
		}
	} else {
		that.Pool.Submit(func() {
			switch errInfo := task.Go(task.Arg); errInfo {
			case nil:
			case errs.ErrEngineShutdown, errs.ErrAcceptSocket:
				that.ErrInfoChan <- errInfo
			default:
				logger.Warningf("error occurs in user-defined function, %v", err)
			}
			PutTask(task)
		})
	}
	that.wg.Done()
	return
}

func (that *Poller) Start(callback iface.IPollCallback) error {
	var wcb sys.WaitCallback = func(fd int, events uint32, trigger bool) (bool, error) {
		var err error
		if !callback.IsBlocked() {
			err = callback.Callback(fd, events)
		}
		if trigger {
			trigger = false
			t := that.priorTasks.Dequeue()
			for ; t != nil; t = that.priorTasks.Dequeue() {
				task := t.(*PollTask)
				that.wg.Add(1)
				that.runTask(task)
			}

			that.wg.Wait()

			select {
			case err = <-that.ErrInfoChan:
				return trigger, err
			default:
				break
			}

			for i := 0; i < iface.MaxTasks; i++ {
				if t = that.tasks.Dequeue(); t == nil {
					break
				}
				task := t.(*PollTask)
				that.wg.Add(1)
				that.runTask(task)
			}

			that.wg.Wait()

			select {
			case err = <-that.ErrInfoChan:
				return trigger, err
			default:
				break
			}

			atomic.StoreInt32(&that.toTrigger, 0)
			if (!that.tasks.IsEmpty() || !that.priorTasks.IsEmpty()) && atomic.CompareAndSwapInt32(&that.toTrigger, 0, 1) {
				if err := sys.Trigger(that.pollEvFd); err == nil {
					trigger = true
				}
			}
		}
		if callback.IsBlocked() {
			err = callback.Callback(fd, events)
		}
		if err != nil {
			return false, err
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

func (that *Poller) AddReadWrite(fd iface.IFd) error {
	return sys.AddReadWrite(that.pollFd, fd.GetFd())
}

func (that *Poller) AddRead(fd iface.IFd) error {
	return sys.AddRead(that.pollFd, fd.GetFd())
}

func (that *Poller) AddWrite(fd iface.IFd) error {
	return sys.AddWrite(that.pollFd, fd.GetFd())
}

func (that *Poller) ModReadWrite(fd iface.IFd) error {
	return sys.ModReadWrite(that.pollFd, fd.GetFd())
}

func (that *Poller) ModRead(fd iface.IFd) error {
	return sys.ModRead(that.pollFd, fd.GetFd())
}

func (that *Poller) ModWrite(fd iface.IFd) error {
	return sys.ModWrite(that.pollFd, fd.GetFd())
}

func (that *Poller) RemoveFd(fd iface.IFd) error {
	return sys.UnRegister(that.pollFd, fd.GetFd())
}
