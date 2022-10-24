package socket

import (
	"errors"
	"syscall"

	"github.com/moqsien/gknet/sys"
	"github.com/moqsien/gknet/utils"
)

var syscallName string = "setsockopt"

func SetKeepAlive(fd, secs int) error {
	if secs <= 0 {
		return errors.New("invalid keep-alive time!")
	}
	err := syscall.SetsockoptInt(fd, sys.SOL_SOCKET, sys.SO_KEEPALIVE, 1)
	if err != nil {
		return utils.SysError(syscallName, err)
	}
	err = syscall.SetsockoptInt(fd, sys.IPPROTO_TCP, sys.TCP_KEEPINTVL, secs)
	if err != nil {
		return utils.SysError(syscallName, err)
	}
	err = syscall.SetsockoptInt(fd, sys.IPPROTO_TCP, sys.TCP_KEEPIDLE, secs)
	return utils.SysError(syscallName, err)
}
