package gkgin

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/moqsien/processes/logger"

	"github.com/moqsien/gknet/gkhttp"
)

var server *gkhttp.GkGin = gkhttp.NewGin()

func Hello(c *gin.Context) {
	logger.Println("[gkgin server]param a received value: ", c.Query("a"))
	c.String(http.StatusOK, "hello gkgin!")
}

func runClient() {
	time.Sleep(3 * time.Second)
	resp, err := http.Get("http://localhost:8082/?a=123")
	if err != nil {
		logger.Println("gkgin client errored: ", err)
		return
	}
	content := make([]byte, 1024)
	resp.Body.Read(content)
	logger.Println("[gkgin client] received content: ", string(content))
	resp.Body.Close()

	time.Sleep(5 * time.Second)
	resp, err = http.Get("http://localhost:8082/?a=123")
	if err != nil {
		logger.Println("gkgin client errored: ", err)
		return
	}
	resp.Body.Read(content)
	logger.Println("[gkgin client] received content: ", string(content))
	resp.Body.Close()
}

func stop() {
	time.Sleep(20 * time.Second)
	fmt.Println("[gkgin stop] close gkgin server.")
	server.Close()
}

func RunGkGin() {
	go runClient()
	go stop()
	server.GET("/", Hello)
	server.Run(":8082")
}
