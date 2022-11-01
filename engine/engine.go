package engine

import (
	"runtime"
	"sync"

	"github.com/moqsien/gknet/balancer"
	"github.com/moqsien/gknet/conn"
	"github.com/moqsien/gknet/eloop"
	"github.com/moqsien/gknet/poll"
	"github.com/moqsien/gknet/socket"
)

const (
	MaxStreamBufferCap = 64 << 10
)

type Engine struct {
	Ln        socket.IListener
	Balancer  eloop.IBalancer
	MainLoop  *eloop.Eloop
	Handler   conn.IEventHandler
	IsClosing int32
	Options   *eloop.Options
	wg        sync.WaitGroup
	cond      *sync.Cond
	once      sync.Once
}

// TODO: reuseport options
func Serve(handler conn.IEventHandler, ln socket.IListener, opt *eloop.Options) (err error) {
	if opt.NumOfLoops <= 0 {
		opt.NumOfLoops = runtime.NumCPU()
	}
	if opt.ReadBuffer <= 0 {
		opt.ReadBuffer = MaxStreamBufferCap
	}
	if opt.WriteBuffer <= 0 {
		opt.WriteBuffer = MaxStreamBufferCap
	}
	engine := new(Engine)
	engine.Ln = ln
	engine.Handler = handler
	engine.Options = opt
	switch opt.LoadBalancer {
	case eloop.RoundRobinLB:
		engine.Balancer = new(balancer.RoundRobin)
	case eloop.LeastConnLB:
		engine.Balancer = new(balancer.LeastConn)
	default:
		engine.Balancer = new(balancer.RoundRobin)
	}
	err = engine.start(opt.NumOfLoops)
	waitchan := make(chan struct{}, 0)
	<-waitchan
	return
}

func (that *Engine) start(numOfLoops int) error {
	return that.startReactors(numOfLoops)
}

func (that *Engine) stop() error {
	return nil
}

func (that *Engine) startReactors(numOfLoops int) error {
	for i := 0; i < numOfLoops; i++ {
		if p, err := poll.New(); err == nil {
			loop := new(eloop.Eloop)
			loop.Listener = that.Ln
			loop.Index = i
			p.Buffer = make([]byte, that.Options.ReadBuffer)
			p.Eloop = loop
			loop.Poller = p
			loop.Engine = that
			loop.ConnList = make(map[int]*conn.Conn)
			that.Balancer.Register(loop)
		} else {
			return err
		}
	}

	// Start sub reactors in background.
	that.startSubReactors()

	if p, err := poll.New(); err == nil {
		loop := new(eloop.Eloop)
		loop.Listener = that.Ln
		loop.Index = -1
		p.Eloop = loop
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
	that.Balancer.Iterator(func(i int, loop *eloop.Eloop) bool {
		that.wg.Add(1)
		go func() {
			loop.StartAsSubLoop(that.Options.LockOSThread)
			that.wg.Done()
		}()
		return true
	})
}

func (that *Engine) GetOptions() *eloop.Options {
	return that.Options
}

func (that *Engine) GetBalancer() eloop.IBalancer {
	return that.Balancer
}

func (that *Engine) GetHandler() conn.IEventHandler {
	return that.Handler
}
