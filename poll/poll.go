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
	pollFd         int             // poll file descriptor
	pollEvFd       int             // poll event file descriptor
	priorTasks     queue.TaskQueue // tasks with priority
	tasks          queue.TaskQueue // tasks
	toTrigger      int32           // atomic number to trigger tasks
	Eloop          iface.IELoop    // eventloop
	Pool           *ants.Pool      // goroutine pool for running tasks
	ErrForStop     chan error      // channel for sending error info to stop the whole engine
	wg             *sync.WaitGroup // wait for tasks to complete
	ReadBufferSize int             // size of read buffer when reading from fd
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

func (that *Poller) runTask(task *PollTask, wg *sync.WaitGroup) (err error) {
	wg.Add(1)
	if that.Pool == nil {
		defer wg.Done()
		switch err = task.Go(task.Arg); err {
		case nil:
			return
		case errs.ErrEngineShutdown, errs.ErrAcceptSocket:
			return err
		default:
			logger.Warningf("error occurs in user-defined function, %v", err)
			return nil
		}
	} else {
		that.Pool.Submit(func() {
			defer wg.Done()
			switch errInfo := task.Go(task.Arg); errInfo {
			case nil:
			case errs.ErrEngineShutdown, errs.ErrAcceptSocket:
				select {
				case that.ErrForStop <- errInfo:
				default:
					break
				}
			default:
				logger.Warningf("error occurs in user-defined function, %v", err)
			}
			PutTask(task)
			return
		})
	}
	return
}

func (that *Poller) Start(callback iface.IPollCallback) error {
	var wcb sys.WaitCallback = func(fd int, events uint32, trigger bool, wg *sync.WaitGroup) (bool, error) {
		var (
			err     error
			errChan chan error
		)
		if !callback.IsBlocked() { // for connection read and write.
			errChan = callback.AsyncWaitCallback(fd, events, wg)
		}

		if trigger {
			trigger = false
			t := that.priorTasks.Dequeue()
			for ; t != nil; t = that.priorTasks.Dequeue() {
				task := t.(*PollTask)
				that.runTask(task, wg)
			}

			for i := 0; i < iface.MaxTasks; i++ {
				if t = that.tasks.Dequeue(); t == nil {
					break
				}
				task := t.(*PollTask)
				that.runTask(task, wg)
			}

			wg.Wait()
			select {
			case err = <-that.ErrForStop:
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
		} else {
			wg.Wait()
		}

		if callback.IsBlocked() { // for accpeting.
			err = callback.Callback(fd, events)
		}

		if errChan != nil {
			select {
			case err = <-errChan:
				return false, err
			default:
				break
			}
		}
		if err != nil {
			return false, err
		}
		return trigger, err
	}
	return sys.WaitPoll(that.pollFd, that.pollEvFd, wcb, doWaitCallbackErr, that.wg)
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
