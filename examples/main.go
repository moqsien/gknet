package main

import (
	"net"
	"time"

	"github.com/moqsien/processes/logger"

	"github.com/moqsien/gknet/conn"
	"github.com/moqsien/gknet/eloop"
	"github.com/moqsien/gknet/engine"
	"github.com/moqsien/gknet/socket"
	"github.com/moqsien/gknet/sys"
)

var Fd int
var connection *conn.Conn

type Server struct{}

func (that *Server) OnOpen(c *conn.Conn) (data []byte, err error) {
	logger.Println("hello onopen")
	return []byte{}, nil
}

func (that *Server) OnTrack(c *conn.Conn) error {
	logger.Println("!!!!!!!!!!ontrack")
	_, err := c.Write([]byte("hello gknet"))
	var content []byte = make([]byte, 30)
	_, err = c.Read(content)
	logger.Println("--content: ", string(content), err)
	return err
}

func (that *Server) OnAccept(c *conn.Conn) error {
	logger.Println("+++pollerFd: ", c.Poller.GetFd(), ", connFd: ", c.Fd)
	Fd = c.Fd
	connection = c
	return nil
}

func (that *Server) OnClose(c *conn.Conn) error {
	return nil
}

func run() {
	ln, _ := socket.Listen("tcp", "127.0.0.1:20000")
	engine.Serve(&Server{}, ln, &eloop.Options{})
}

func client() {
	time.Sleep(2 * time.Second)
	conn, _ := net.Dial("tcp", "127.0.0.1:20000")
	// n, err := conn.Write([]byte("hello-----"))
	// logger.Println("++write: ", err, ", byte: ", n)
	// content := make([]byte, 1024)
	// conn.Read(content)
	// logger.Println("&&&received: ", string(content))
	conn.Close()
	sys.SocketClosed(Fd)
	time.Sleep(30 * time.Second)
	logger.Println("=====================================")
	// sys.CloseFd(Fd)
	logger.Println("-------fd in client: ", Fd)
	sys.SocketClosed(Fd)
	logger.Println(connection)
}

func main() {
	go client()
	run()
}
