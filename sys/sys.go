package sys

type WaitCallback func(fd int, events int64, trigger bool) error

const (
	InitPollSize = 64
	EVFilterFd   = -0xd
)
