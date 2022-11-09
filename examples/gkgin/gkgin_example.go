package gkgin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/moqsien/processes/logger"

	"github.com/moqsien/gknet/gkhttp"
)

var server *gkhttp.GkGin = gkhttp.NewGin()

func Hello(c *gin.Context) {
	c.String(http.StatusOK, "hello gkgin!")
}

func runClient() {
	time.Sleep(3 * time.Second)
	resp, err := http.Get("http://localhost:8082/")
	if err != nil {
		logger.Println("gkgin client errored: ", err)
		return
	}
	defer resp.Body.Close()
	content := make([]byte, 1024)
	resp.Body.Read(content)
	logger.Println("[gkgin client] received content: ", string(content))
}

func RunGkGin() {
	go runClient()
	server.GET("/", Hello)
	server.Run(":8082")
}
