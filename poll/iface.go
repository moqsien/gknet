package poll

type IELoop interface {
	AddConnCount(i int32)
	RemoveConn(fd int)
}

type IFd interface {
	GetFd() int
}
