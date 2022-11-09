package gktcp

import (
	"net"
	"time"

	"github.com/moqsien/processes/logger"

	"github.com/moqsien/gknet/conn"
	"github.com/moqsien/gknet/engine"
	"github.com/moqsien/gknet/iface"
	"github.com/moqsien/gknet/socket"
)

type GkTCPHandler struct{}

func (that *GkTCPHandler) OnAccept(c iface.RawConn) error {
	connection := c.(*conn.Conn)
	logger.Println("[OnAccept] Fd: ", connection.Fd)
	return nil
}

func (that *GkTCPHandler) OnOpen(c *iface.Context) (data []byte, err error) {
	connection := c.RawConn.(*conn.Conn)
	logger.Println("[OnOpen] Fd: ", connection.Fd)
	return nil, nil
}

func (that *GkTCPHandler) OnClose(c *iface.Context) error {
	connection := c.RawConn.(*conn.Conn)
	logger.Println("[OnClose] Fd: ", connection.Fd)
	return nil
}

func (that *GkTCPHandler) OnTrack(c *iface.Context) (err error) {
	connection := c.RawConn.(*conn.Conn)
	logger.Println("[Ontrack] Fd: ", connection.Fd)
	_, err = c.Write([]byte("hello gknet, this is a tcp example!"))
	content := make([]byte, 100)
	_, err = c.Read(content)
	logger.Println("[Ontrack] Fd: ", connection.Fd, ", received content from client: ", content)
	return
}

func runServer() {
	ln, _ := socket.Listen("tcp", "127.0.0.1:20000")
	eng := engine.New()
	eng.Serve(&GkTCPHandler{}, ln, &iface.Options{})
}

func runClient() {
	conn, _ := net.Dial("tcp", "127.0.0.1:20000")
	conn.Write([]byte("hello-----"))
	content := make([]byte, 1024)
	conn.Read(content)
	logger.Println("&&&received: ", string(content))
	conn.Close()
	time.Sleep(10 * time.Second)
}

func RunTcp() {
	go runClient()
	runServer()
}
