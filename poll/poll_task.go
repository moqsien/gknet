package poll

import (
	"sync"

	"github.com/moqsien/gknet/iface"
)

type PollTask struct {
	Go  iface.PollTaskFunc
	Arg iface.PollTaskArg
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
