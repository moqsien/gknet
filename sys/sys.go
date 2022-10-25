package sys

import "syscall"

type WaitCallback func(fd int, events int64, trigger bool) (newTrigger bool, err error)

type DoError func(err error) error

const (
	MaxPollSize  = 1024
	MinPollSize  = 32
	InitPollSize = 128
	EVFilterFd   = -0xd
)

func Close(fd int) error {
	return syscall.Close(fd)
}
