/*
Reactor engine.
*/
package engine

import (
	"net"
	"runtime"
	"sync"

	"github.com/moqsien/processes/logger"
	"github.com/panjf2000/ants/v2"

	"github.com/moqsien/gknet/balancer"
	"github.com/moqsien/gknet/eloop"
	"github.com/moqsien/gknet/iface"
	"github.com/moqsien/gknet/poll"
	"github.com/moqsien/gknet/utils/errs"
)

type Engine struct {
	Listener  iface.IListener
	Balancer  iface.IBalancer
	MainLoop  *eloop.Eloop
	Handler   iface.IEventHandler
	IsClosing int32
	Options   *iface.Options
	Pool      *ants.Pool
	wg        sync.WaitGroup
	cond      *sync.Cond
	once      sync.Once
}

func New() *Engine {
	return new(Engine)
}

// TODO: reuseport options
func (that *Engine) Serve(handler iface.IEventHandler, ln iface.IListener, opt *iface.Options) (err error) {
	if opt.NumOfLoops <= 0 {
		opt.NumOfLoops = runtime.NumCPU()
	}
	if opt.ReadBuffer <= 0 {
		opt.ReadBuffer = iface.MaxStreamBufferCap
	}
	if opt.WriteBuffer <= 0 {
		opt.WriteBuffer = iface.MaxStreamBufferCap
	}
	if opt.GoroutineSize <= 0 {
		opt.GoroutineSize = iface.DefaultGoroutineSize
	}
	that.Listener = ln
	that.Handler = handler
	that.Options = opt
	that.Pool, err = ants.NewPool(opt.GoroutineSize)
	if err != nil {
		logger.Println(err)
		return err
	}
	that.cond = sync.NewCond(&sync.Mutex{})
	that.wg = sync.WaitGroup{}
	that.once = sync.Once{}
	switch opt.LoadBalancer {
	case iface.RoundRobinLB:
		that.Balancer = new(balancer.RoundRobin)
	case iface.LeastConnLB:
		that.Balancer = new(balancer.LeastConn)
	default:
		that.Balancer = new(balancer.RoundRobin)
	}
	err = that.start(opt.NumOfLoops)
	defer that.stop()
	return
}

func (that *Engine) start(numOfLoops int) error {
	return that.startReactors(numOfLoops)
}

// send stop signal to the engine.
func (that *Engine) Stop() {
	that.once.Do(func() {
		that.cond.L.Lock()
		that.cond.Signal()
		that.cond.L.Unlock()
	})
}

func (that *Engine) waitForStopSignal() {
	that.cond.L.Lock()
	that.cond.Wait()
	that.cond.L.Unlock()
}

func (that *Engine) stop() (err error) {
	// wait until that.Stop() is called.
	that.waitForStopSignal()

	// close all connections.
	that.Balancer.Iterator(func(key int, val iface.IELoop) bool {
		err := val.GetPoller().AddPriorTask(func(_ iface.PollTaskArg) error { return errs.ErrEngineShutdown }, nil)
		if err != nil {
			logger.Errorf("failed to call UrgentTrigger on sub event-loop when stopping engine: %v", err)
		}
		return true
	})

	if that.MainLoop != nil {
		that.Listener.Close()
		that.MainLoop.Poller.AddPriorTask(func(_ iface.PollTaskArg) error { return errs.ErrEngineShutdown }, nil)
	}

	// wait for all connections to close.
	that.wg.Wait()

	// close all pollers.
	that.Balancer.Iterator(func(key int, val iface.IELoop) bool {
		val.GetPoller().Close()
		return true
	})

	if that.MainLoop != nil {
		err = that.MainLoop.Poller.Close()
	}
	that.Pool.Release()
	return err
}

func (that *Engine) startReactors(numOfLoops int) error {
	for i := 0; i < numOfLoops; i++ {
		if p, err := poll.New(); err == nil {
			loop := new(eloop.Eloop)
			loop.Listener = that.Listener
			loop.Index = i
			p.ReadBufferSize = that.Options.ReadBuffer
			p.Eloop = loop
			p.ErrForStop = make(chan error, 2)
			p.Pool = that.Pool
			p.AsyncReadWriteFd = that.Options.AsyncReadWriteFd
			loop.Poller = p
			loop.Engine = that
			loop.ConnList = make(map[int]net.Conn)
			that.Balancer.Register(loop)
		} else {
			return err
		}
	}

	// Start sub reactors in background.
	that.startSubReactors()

	if p, err := poll.New(); err == nil {
		loop := new(eloop.Eloop)
		loop.Listener = that.Listener
		loop.Index = -1
		p.ReadBufferSize = that.Options.ReadBuffer
		p.Eloop = loop
		p.ErrForStop = make(chan error, 2)
		p.Pool = that.Pool
		loop.Poller = p
		loop.Engine = that
		if err = loop.Poller.AddRead(loop.Listener); err != nil {
			return err
		}
		that.MainLoop = loop

		// Start main reactor in background.
		that.wg.Add(1)
		go func() {
			loop.StartAsMainLoop(that.Options.LockOSThread)
			that.wg.Done()
		}()
	} else {
		return err
	}
	return nil
}

func (that *Engine) startSubReactors() {
	that.Balancer.Iterator(func(i int, loop iface.IELoop) bool {
		that.wg.Add(1)
		go func() {
			loop.StartAsSubLoop(that.Options.LockOSThread)
			that.wg.Done()
		}()
		return true
	})
}

func (that *Engine) GetOptions() *iface.Options {
	return that.Options
}

func (that *Engine) GetBalancer() iface.IBalancer {
	return that.Balancer
}

func (that *Engine) GetHandler() iface.IEventHandler {
	return that.Handler
}
