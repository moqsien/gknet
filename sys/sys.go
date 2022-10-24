package sys

import "sync"

type GkEvent struct {
	Fd    uint64
	Event int64
}

type GkEventList []*GkEvent

var GkEPool = &sync.Pool{
	New: func() interface{} {
		return &GkEvent{}
	},
}

func GkGet() *GkEvent {
	return GkEPool.Get().(*GkEvent)
}

func GkPut(ev *GkEvent) {
	GkEPool.Put(ev)
}
