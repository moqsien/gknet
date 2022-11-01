package sys

import (
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/moqsien/processes/logger"

	"github.com/moqsien/gknet/utils"
)

type EventHandler interface {
	WriteToFd() error
	ReadFromFd() error
	Close() error
}

type WaitCallback func(fd int, events uint32, trigger bool) (newTrigger bool, err error)

type DoError func(err error) error

const (
	MaxPollSize         = 1024
	MinPollSize         = 32
	InitPollSize        = 128
	EVFilterFd          = -0xd
	DefaultTCPKeepAlive = 15 // Seconds
)

func CloseFd(fd int) error {
	return utils.SysError("fd_close", syscall.Close(fd))
}

func transKeepAlive(t ...time.Duration) (secs int) {
	if len(t) == 0 {
		return DefaultTCPKeepAlive
	}
	secs = int(t[0] / time.Second)
	if secs >= 1 {
		return
	}
	return DefaultTCPKeepAlive
}

func SetKeepAlive(fd int, timeout ...time.Duration) (err error) {
	// timeout in seconds.
	secs := transKeepAlive(timeout...)
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, SO_KEEPALIVE, 1)
	if err != nil {
		return utils.SysError("setsockopt", err)
	}
	err = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_KEEPINTVL, secs)
	if err != nil {
		return utils.SysError("setsockopt", err)
	}
	err = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_KEEPIDLE, secs)
	runtime.KeepAlive(fd)
	return utils.SysError("setsockopt", err)
}

func SetReusePort(fd int) error {
	return utils.SysError("setsockopt", syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, SO_REUSEPORT, 1))
}

func SetReuseAddr(fd int) error {
	return utils.SysError("setsockopt", syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1))
}

func SetRecvBufferSize(fd, size int) error {
	return utils.SysError("setsockopt", syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF, size))
}

func SetSendBufferSize(fd, size int) error {
	return utils.SysError("setsockopt", syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_SNDBUF, size))
}

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

func HandleEvents(events uint32, handler EventHandler) (err error) {
	if events&ClosedFdEvents != 0 {
		err = handler.Close()
		return
	}

	if events&OutEvents != 0 {
		err = handler.WriteToFd()
		if err != nil {
			return
		}
	}

	if events&InEvents != 0 {
		err = handler.ReadFromFd()
		logger.Println("**&&&&err: ", err)
		if err != nil {
			return
		}
	}
	return
}

var _zero uintptr

func bytes2iovec(bs [][]byte) []syscall.Iovec {
	iovecs := make([]syscall.Iovec, len(bs))
	for i, b := range bs {
		iovecs[i].SetLen(len(b))
		if len(b) > 0 {
			iovecs[i].Base = &b[0]
		} else {
			iovecs[i].Base = (*byte)(unsafe.Pointer(&_zero))
		}
	}
	return iovecs
}

func writev(fd int, iovs []syscall.Iovec) (n int, err error) {
	var _p0 unsafe.Pointer
	if len(iovs) > 0 {
		_p0 = unsafe.Pointer(&iovs[0])
	} else {
		_p0 = unsafe.Pointer(&_zero)
	}
	r0, _, e1 := syscall.Syscall(syscall.SYS_WRITEV, uintptr(fd), uintptr(_p0), uintptr(len(iovs)))
	n = int(r0)
	err = errnoErr(e1)
	return
}

func readv(fd int, iovs []syscall.Iovec) (n int, err error) {
	var _p0 unsafe.Pointer
	if len(iovs) > 0 {
		_p0 = unsafe.Pointer(&iovs[0])
	} else {
		_p0 = unsafe.Pointer(&_zero)
	}
	r0, _, e1 := syscall.Syscall(syscall.SYS_READV, uintptr(fd), uintptr(_p0), uintptr(len(iovs)))
	n = int(r0)
	err = errnoErr(e1)
	return
}

func Writev(fd int, iovs [][]byte) (n int, err error) {
	iovecs := bytes2iovec(iovs)
	n, err = writev(fd, iovecs)
	return n, err
}

func Readv(fd int, iovs [][]byte) (n int, err error) {
	iovecs := bytes2iovec(iovs)
	n, err = readv(fd, iovecs)
	return n, err
}

func Write(fd int, p []byte) (n int, err error) {
	return syscall.Write(fd, p)
}

func Read(fd int, p []byte) (n int, err error) {
	return syscall.Read(fd, p)
}

func WriteUdp(fd int, p []byte, flags int, to syscall.Sockaddr) (err error) {
	return syscall.Sendto(fd, p, flags, to)
}
