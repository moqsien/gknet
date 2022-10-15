package poll

type IEPool interface {
	AddConnCount(i int32)
	RemoveConn(fd int)
}
