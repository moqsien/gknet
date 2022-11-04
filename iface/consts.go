package iface

const (
	ConnAsyncWriteAdapter  ConnAdapter = 0
	ConnNoneAdapter        ConnAdapter = 1
	ConnWritevAdapter      ConnAdapter = 2
	ConnAsyncWritevAdapter ConnAdapter = 3
)

const (
	RoundRobinLB       int = 0
	LeastConnLB        int = 1
	MaxStreamBufferCap int = 64 << 10
	IovMax             int = 1024
	MaxTasks           int = 256
)
