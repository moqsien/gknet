package gkhttps

import (
	"net/http"
	"time"

	"github.com/moqsien/processes/logger"

	"github.com/moqsien/gknet/gkhttp"
)

var handler *http.ServeMux = http.NewServeMux()

func Hello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello gkhttp!"))
}

func runClient() {
	time.Sleep(3 * time.Second)
	resp, err := http.Get("http://localhost:8083/")
	if err != nil {
		logger.Println("gkhttp client errored: ", err)
		return
	}
	defer resp.Body.Close()
	content := make([]byte, 1024)
	resp.Body.Read(content)
	logger.Println("[gkhttp client] received content: ", string(content))
}

func RunHttp() {
	go runClient()
	handler.HandleFunc("/", Hello)
	server := gkhttp.NewHttpServer(handler)
	server.ListenAndServe("tcp", ":8083")
}
