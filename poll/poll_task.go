package poll

import "sync"

type PollTaskArg interface{}

type PollTaskFunc func(arg PollTaskArg) error

type PollTask struct {
	Go  PollTaskFunc
	Arg PollTaskArg
}

var PollTaskPool = sync.Pool{
	New: func() interface{} {
		return &PollTask{}
	},
}

func PutTask(t *PollTask) {
	t.Go, t.Arg = nil, nil
	PollTaskPool.Put(t)
}

func GetTask() *PollTask {
	return PollTaskPool.Get().(*PollTask)
}
