package gkhttp

import (
	"bufio"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

const (
	reqContentLength = "Content-Length"
	host             = "Host"
)

var bodyPool = sync.Pool{
	New: func() interface{} {
		return &body{}
	},
}

func freeBody(b *body) {
	*b = body{}
	bodyPool.Put(b)
}

var cancelPool = sync.Pool{
	New: func() interface{} {
		return make(<-chan struct{}, 1)
	},
}

func freeCancel(c <-chan struct{}) {
	select {
	case <-c:
	default:
	}
	cancelPool.Put(c)
}

var reqHeaderPool = sync.Pool{
	New: func() interface{} {
		return make(http.Header)
	},
}

func reqFreeHeader(h http.Header) {
	if h == nil {
		return
	}
	for key := range h {
		h.Del(key)
	}
	reqHeaderPool.Put(h)
}

var requestPool = sync.Pool{
	New: func() interface{} {
		return &http.Request{}
	},
}

// FreeRequest frees the request.
func FreeRequest(r *http.Request) {
	if r == nil {
		return
	}
	reqFreeHeader(r.Header)
	if body, ok := r.Body.(*body); ok {
		freeBody(body)
	}
	freeCancel(r.Cancel)
	*r = http.Request{}
	requestPool.Put(r)
}

// ReadRequest reads and parses an incoming request from b.
//
// ReadRequest is a low-level function and should only be used for
// specialized applications; most code should use the Server to read
// requests and handle them via the Handler interface. ReadRequest
// only supports HTTP/1.x requests.
func ReadRequest(b *bufio.Reader) (*http.Request, error) {
	return http.ReadRequest(b)
}

// ReadFastRequest is like ReadRequest but with the simple request parser.
func ReadFastRequest(b *bufio.Reader) (*http.Request, error) {
	statusLineData, err := readLine(b)
	if err != nil {
		return nil, err
	}
	statusLine := *(*string)(unsafe.Pointer(&statusLineData))
	strs := strings.Split(statusLine, " ")
	if len(strs) != 3 {
		return nil, errors.New("status line error ")
	}
	req := requestPool.Get().(*http.Request)
	req.Header = reqHeaderPool.Get().(http.Header)
	req.Method = strs[0]
	URL, err := url.Parse(strs[1])
	if err != nil {
		return nil, err
	}
	req.URL = URL
	req.Proto = strs[2]
	for {
		lineData, err := readLine(b)
		if err != nil {
			return nil, err
		}
		line := *(*string)(unsafe.Pointer(&lineData))
		if len(line) > 0 {
			strs := strings.Split(line, " ")
			key := strs[0][:len(strs[0])-1]
			req.Header[key] = strs[1:]
		} else {
			break
		}
	}
	if v, ok := req.Header[host]; ok {
		req.Host = v[0]
	}
	if v, ok := req.Header[reqContentLength]; ok {
		ContentLength, _ := strconv.ParseInt(v[0], 0, 64)
		req.ContentLength = ContentLength
	}
	body := bodyPool.Get().(*body)
	body.i = 0
	body.l = req.ContentLength
	body.b = b
	req.Body = body
	req.Cancel = cancelPool.Get().(<-chan struct{})
	return req, nil
}

type body struct {
	b *bufio.Reader
	i int64 // current reading index
	l int64 // content length
}

// Read implements the io.Reader interface.
func (b *body) Read(p []byte) (n int, err error) {
	if b.i >= b.l {
		return 0, io.EOF
	}
	if b.i+int64(len(p)) <= b.l {
		n, err = b.b.Read(p)
		b.i += int64(n)
		return
	}
	n, err = b.b.Read(p[:b.l-b.i])
	b.i += int64(n)
	return
}

// Read implements the io.Closer interface.
func (b *body) Close() error {
	if b.l <= 0 {
		return nil
	}
	if b.i >= b.l {
		return nil
	}
	io.CopyN(ioutil.Discard, b, b.l-b.i)
	return nil
}

func readLine(b *bufio.Reader) ([]byte, error) {
	line, isPrefix, err := b.ReadLine()
	if !isPrefix {
		return line, err
	}
	for {
		fragment, isPrefix, err := b.ReadLine()
		line = append(line, fragment...)
		if !isPrefix {
			return line, err
		}
	}
}
