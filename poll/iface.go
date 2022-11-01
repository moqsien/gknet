package poll

type IELoop interface {
	AddConnCount(i int32)
	RemoveConn(fd int)
}

type IFd interface {
	GetFd() int
}

type PollCallback interface {
	Callback(fd int, events uint32) error
	IsBlocked() bool
}
