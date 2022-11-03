package poll

import "net"

type IELoop interface {
	AddConnCount(i int32) int32
	RemoveConn(fd int)
	GetConnList() map[int]net.Conn
}

type IFd interface {
	GetFd() int
}

type IPollCallback interface {
	Callback(fd int, events uint32) error
	IsBlocked() bool
}
