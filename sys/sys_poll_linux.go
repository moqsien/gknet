//go:build linux

package sys

import (
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/moqsien/processes/logger"

	"github.com/moqsien/gknet/utils"
)

var ePool = &sync.Pool{New: func() any {
	return &syscall.EpollEvent{}
}}

func eGet() *syscall.EpollEvent {
	return ePool.Get().(*syscall.EpollEvent)
}

func ePut(event *syscall.EpollEvent) {
	ePool.Put(event)
}

const (
	ReadEvents      = syscall.EPOLLPRI | syscall.EPOLLIN
	WriteEvents     = syscall.EPOLLOUT
	ReadWriteEvents = ReadEvents | WriteEvents
)

var eSysName map[int]string = map[int]string{
	syscall.EPOLL_CTL_ADD: "epoll_ctl_add",
	syscall.EPOLL_CTL_MOD: "epoll_ctl_mod",
	syscall.EPOLL_CTL_DEL: "epoll_ctl_del",
}

func epollFdHandler(pollFd, fd, ctlAction int, evs uint32) (err error) {
	var event *syscall.EpollEvent
	if ctlAction != syscall.EPOLL_CTL_DEL {
		event = eGet()
		defer ePut(event)
		event.Fd, event.Events = int32(fd), evs
	}
	err = syscall.EpollCtl(pollFd, ctlAction, fd, event)
	return utils.SysError(eSysName[ctlAction], err)
}

func AddReadWrite(pollFd, fd int) (err error) {
	return epollFdHandler(pollFd, fd, syscall.EPOLL_CTL_ADD, ReadWriteEvents)
}

func AddRead(pollFd, fd int) (err error) {
	return epollFdHandler(pollFd, fd, syscall.EPOLL_CTL_ADD, ReadEvents)
}

func AddWrite(pollFd, fd int) (err error) {
	return epollFdHandler(pollFd, fd, syscall.EPOLL_CTL_ADD, WriteEvents)
}

func ModRead(pollFd, fd int) (err error) {
	return epollFdHandler(pollFd, fd, syscall.EPOLL_CTL_MOD, ReadEvents)
}

func ModWrite(pollFd, fd int) (err error) {
	return epollFdHandler(pollFd, fd, syscall.EPOLL_CTL_MOD, WriteEvents)
}

func ModReadWrite(pollFd, fd int) (err error) {
	return epollFdHandler(pollFd, fd, syscall.EPOLL_CTL_MOD, ReadWriteEvents)
}

func UnRegister(pollFd, fd int) (err error) {
	return epollFdHandler(pollFd, fd, syscall.EPOLL_CTL_DEL, 0)
}

func expand(size int) (newSize int, events []syscall.EpollEvent) {
	newSize = size << 1
	events = make([]syscall.EpollEvent, newSize)
	return
}

func shrink(size int) (newSize int, events []syscall.EpollEvent) {
	newSize = size >> 1
	events = make([]syscall.EpollEvent, newSize)
	return
}

func WaitPoll(pollFd, pollEvFd int, w WaitCallback, doCallbackErr DoError, wg *sync.WaitGroup) error {
	size := InitPollSize
	events := make([]syscall.EpollEvent, size)
	var (
		trigger      bool
		timeout      int = -1
		pollEvBuffer     = make([]byte, 8)
	)
	for {
		n, err := syscall.EpollWait(pollFd, events, timeout)
		if n == 0 || (n < 0 && err == syscall.EINTR) {
			timeout = -1
			runtime.Gosched()
			continue
		} else if err != nil {
			logger.Errorf("error occurs in epoll: %v", utils.SysError("epoll_wait", err))
			return err
		}
		timeout = 0
		for i := 0; i < n; i++ {
			ev := &events[i]
			fd := int(ev.Fd)
			if fd == pollEvFd {
				trigger = true
				syscall.Read(pollEvFd, pollEvBuffer)
			}
			if i == n-1 {
				trigger, err = w(fd, ev.Events, trigger, wg)
			} else {
				trigger, err = w(fd, ev.Events, false, wg)
			}
			err = doCallbackErr(err)
			if err != nil {
				return err
			}
		}

		if n == size && (size<<1 <= MaxPollSize) {
			size, events = expand(size)
		} else if (n < size>>1) && (size>>1 >= MinPollSize) {
			size, events = shrink(size)
		}
	}
}

func pEventFd(initval uint, flags int) (fd int, err error) {
	r0, _, e1 := syscall.Syscall(syscall.SYS_EVENTFD2, uintptr(initval), uintptr(flags), 0)
	fd, err = int(r0), errnoErr(e1)
	return
}

func CreatePoll() (pollFd, pollEvFd int, err error) {
	pollFd, err = syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		err = utils.SysError("epoll_create1", err)
		return
	}
	pollEvFd, err = pEventFd(0, syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC)
	if err != nil {
		syscall.Close(pollFd)
		err = utils.SysError("epoll_eventfd", err)
		return
	}
	err = AddRead(pollFd, pollEvFd)
	if err != nil {
		syscall.Close(pollFd)
		syscall.Close(pollEvFd)
		err = utils.SysError("epoll_eventfd_add", err)
		return
	}
	return
}

var (
	u uint64 = 1
	b        = (*(*[8]byte)(unsafe.Pointer(&u)))[:]
)

func Trigger(pollEvFd int) (err error) {
	if _, err = syscall.Write(pollEvFd, b); err == syscall.EAGAIN {
		err = nil
	}
	return utils.SysError("pollEvFd_write", err)
}

func Accept(listenerFd int, timeout ...time.Duration) (int, syscall.Sockaddr, error) {
	nfd, sock, err := syscall.Accept4(listenerFd, syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC)
	switch err {
	case nil:
		return nfd, sock, err
	default:
		return -1, nil, err
	case syscall.ENOSYS:
	case syscall.EINVAL:
	case syscall.EACCES:
	case syscall.EFAULT:
	}
	nfd, sock, err = syscall.Accept(listenerFd)
	if err == nil {
		syscall.CloseOnExec(nfd)
	} else {
		return -1, nil, err
	}
	if err = syscall.SetNonblock(nfd, true); err != nil {
		syscall.Close(nfd)
		return -1, nil, err
	}
	SetKeepAlive(nfd, timeout...)
	return nfd, sock, nil
}

/*fd state*/
type state struct {
	State         uint8   `name:"tcpinfo_state" help:"TCP state"`
	CaState       uint8   `name:"tcpinfo_ca_state" help:"state of congestion avoidance"`
	Retransmits   uint8   `name:"tcpinfo_retransmits" help:"number of retranmissions on timeout invoked"`
	Probes        uint8   `name:"tcpinfo_probes" help:"consecutive zero window probes that have gone unanswered"`
	Backoff       uint8   `name:"tcpinfo_backoff" help:"used for exponential backoff re-transmission"`
	Options       uint8   `name:"tcpinfo_options" help:"number of requesting options"`
	pad           [2]byte `unexported:"true"`
	Rto           uint32  `name:"tcpinfo_rto" help:"tcp re-transmission timeout value, the unit is microsecond"`
	Ato           uint32  `name:"tcpinfo_ato" help:"ack timeout, unit is microsecond"`
	SndMss        uint32  `name:"tcpinfo_snd_mss" help:"current maximum segment size"`
	RcvMss        uint32  `name:"tcpinfo_rcv_mss" help:"maximum observed segment size from the remote host"`
	Unacked       uint32  `name:"tcpinfo_unacked" help:"number of unack'd segments"`
	Sacked        uint32  `name:"tcpinfo_sacked" help:"scoreboard segment marked SACKED by sack blocks accounting for the pipe algorithm"`
	Lost          uint32  `name:"tcpinfo_lost" help:"scoreboard segments marked lost by loss detection heuristics accounting for the pipe algorithm"`
	Retrans       uint32  `name:"tcpinfo_retrans" help:"how many times the retran occurs"`
	Fackets       uint32  `name:"tcpinfo_fackets" help:""`
	LastDataSent  uint32  `name:"tcpinfo_last_data_sent" help:"time since last data segment was sent"`
	LastAckSent   uint32  `name:"tcpinfo_last_ack_sent" help:"how long time since the last ack sent"`
	LastDataRecv  uint32  `name:"tcpinfo_last_data_recv" help:"time since last data segment was received"`
	LastAckRecv   uint32  `name:"tcpinfo_last_ack_recv" help:"how long time since the last ack received"`
	Pmtu          uint32  `name:"tcpinfo_path_mtu" help:"path MTU"`
	RcvSsthresh   uint32  `name:"tcpinfo_rev_ss_thresh" help:"tcp congestion window slow start threshold"`
	Rtt           uint32  `name:"tcpinfo_rtt" help:"smoothed round trip time"`
	Rttvar        uint32  `name:"tcpinfo_rtt_var" help:"RTT variance"`
	SndSsthresh   uint32  `name:"tcpinfo_snd_ss_thresh" help:"slow start threshold"`
	SndCwnd       uint32  `name:"tcpinfo_snd_cwnd" help:"congestion window size"`
	Advmss        uint32  `name:"tcpinfo_adv_mss" help:"advertised maximum segment size"`
	Reordering    uint32  `name:"tcpinfo_reordering" help:"number of reordered segments allowed"`
	RcvRtt        uint32  `name:"tcpinfo_rcv_rtt" help:"receiver side RTT estimate"`
	RcvSpace      uint32  `name:"tcpinfo_rcv_space" help:"space reserved for the receive queue"`
	TotalRetrans  uint32  `name:"tcpinfo_total_retrans" help:"total number of segments containing retransmitted data"`
	PacingRate    uint64  `name:"tcpinfo_pacing_rate" help:"the pacing rate"`
	maxPacingRate uint64  `name:"tcpinfo_max_pacing_rate" help:"" unexported:"true"`
	BytesAcked    uint64  `name:"tcpinfo_bytes_acked" help:"bytes acked"`
	BytesReceived uint64  `name:"tcpinfo_bytes_received" help:"bytes received"`
	SegsOut       uint32  `name:"tcpinfo_segs_out" help:"segments sent out"`
	SegsIn        uint32  `name:"tcpinfo_segs_in" help:"segments received"`
	NotsentBytes  uint32  `name:"tcpinfo_notsent_bytes" help:""`
	MinRtt        uint32  `name:"tcpinfo_min_rtt" help:""`
	DataSegsIn    uint32  `name:"tcpinfo_data_segs_in" help:"RFC4898 tcpEStatsDataSegsIn"`
	DataSegsOut   uint32  `name:"tcpinfo_data_segs_out" help:"RFC4898 tcpEStatsDataSegsOut"`
	DeliveryRate  uint64  `name:"tcpinfo_delivery_rate" help:""`
	BusyTime      uint64  `name:"tcpinfo_busy_time" help:"time (usec) busy sending data"`
	RwndLimited   uint64  `name:"tcpinfo_rwnd_limited" help:"time (usec) limited by receive window"`
	SndbufLimited uint64  `name:"tcpinfo_sndbuf_limited" help:"time (usec) limited by send buffer"`
	Delivered     uint32  `name:"tcpinfo_delivered" help:""`
	DeliveredCe   uint32  `name:"tcpinfo_delivered_ce" help:""`
	BytesSent     uint64  `name:"tcpinfo_bytes_sent" help:""`
	BytesRetrans  uint64  `name:"tcpinfo_bytes_retrans" help:"RFC4898 tcpEStatsPerfOctetsRetrans"`
	DsackDups     uint32  `name:"tcpinfo_dsack_dups" help:"RFC4898 tcpEStatsStackDSACKDups"`
	ReordSeen     uint32  `name:"tcpinfo_reord_seen" help:"reordering events seen"`
	RcvOoopack    uint32  `name:"tcpinfo_rcv_ooopack" help:"out-of-order packets received"`
	SndWnd        uint32  `name:"tcpinfo_snd_wnd" help:""`

	TCPCongesAlg string `help:"TCP network congestion-avoidance algorithm"`

	HTTPStatusCode int   `name:"http_status_code" help:"HTTP 1xx-5xx status code"`
	HTTPRcvdBytes  int64 `name:"http_rcvd_bytes" help:"HTTP bytes received"`
	HTTPRequest    int64 `name:"http_request" help:"HTTP request, the unit is microsecond"`
	HTTPResponse   int64 `name:"http_response" help:"HTTP response, the unit is microsecond"`

	DNSResolve   int64 `name:"dns_resolve" help:"domain lookup, the unit is microsecond"`
	TCPConnect   int64 `name:"tcp_connect" help:"TCP connect, the unit is microsecond"`
	TLSHandshake int64 `name:"tls_handshake" help:"TLS handshake, the unit is microsecond"`

	TCPConnectError int64 `name:"tcp_connect_error" help:"total TCP connect error" kind:"counter"`
	DNSResolveError int64 `name:"dns_resolve_error" help:"total DNS resolve error" kind:"counter"`
}

func SocketClosed(fd int) bool {
	size := uint32(232)
	s := &state{}
	syscall.Syscall6(syscall.SYS_GETSOCKOPT, uintptr(fd), syscall.SOL_TCP, syscall.TCP_INFO,
		uintptr(unsafe.Pointer(s)), uintptr(unsafe.Pointer(&size)), 0)
	if s.State == 1 {
		return false
	}
	return true
}
