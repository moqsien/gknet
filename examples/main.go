package main

import (
	"net"
	"time"

	"github.com/moqsien/processes/logger"

	"github.com/moqsien/gknet/conn"
	"github.com/moqsien/gknet/engine"
	"github.com/moqsien/gknet/iface"
	"github.com/moqsien/gknet/socket"
)

var Fd int
var connection *conn.Conn

type Server struct{}

func (that *Server) OnOpen(c *iface.Context) (data []byte, err error) {
	logger.Println("hello onopen")
	return []byte{}, nil
}

func (that *Server) OnTrack(c *iface.Context) error {
	logger.Println("!!!!!!!!!!ontrack")
	_, err := c.Write([]byte("hello gknet"))
	var content []byte = make([]byte, 30)
	_, err = c.Read(content)
	logger.Println("--content: ", string(content), err)
	return err
}

func (that *Server) OnAccept(c iface.RawConn) error {
	co := c.(*conn.Conn)
	logger.Println("Connection Fd: ", co.Fd)
	return nil
}

func (that *Server) OnClose(c *iface.Context) error {
	return nil
}

func run() {
	ln, _ := socket.Listen("tcp", "127.0.0.1:20000")
	eng := engine.New()
	eng.Serve(&Server{}, ln, &iface.Options{})
}

func client() {
	conn, _ := net.Dial("tcp", "127.0.0.1:20000")
	n, err := conn.Write([]byte("hello-----"))
	logger.Println("++write: ", err, ", byte: ", n)
	content := make([]byte, 1024)
	conn.Read(content)
	logger.Println("&&&received: ", string(content))
	conn.Close()
	time.Sleep(10 * time.Second)
	logger.Println("=====================================")
	// sys.CloseFd(Fd)
	logger.Println("-------fd in client: ", Fd)
	logger.Println(connection)
}

func main() {
	go client()
	run()
}
