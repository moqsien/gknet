package gkhttp

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const (
	httpVersion        = "HTTP/1.1 "
	chunk              = "%x\r\n"
	contentLength      = "Content-Length"
	transferEncoding   = "Transfer-Encoding"
	contentType        = "Content-Type"
	date               = "Date"
	connection         = "Connection"
	chunked            = "chunked"
	defaultContentType = "text/plain; charset=utf-8"
	head               = "HEAD"
	emptyString        = ""
)

var (
	buffers           = sync.Map{}
	bufioWriters      = sync.Map{}
	bufioReaderPool   sync.Pool
	assignBuffer      int32
	assignBufioWriter int32
)

func assignBufferPool(size int) *sync.Pool {
	for {
		if p, ok := buffers.Load(size); ok {
			return p.(*sync.Pool)
		}
		if atomic.CompareAndSwapInt32(&assignBuffer, 0, 1) {
			var pool = &sync.Pool{New: func() interface{} {
				return make([]byte, size)
			}}
			buffers.Store(size, pool)
			atomic.StoreInt32(&assignBuffer, 0)
			return pool
		}
	}
}

func assignBufioWriterPool(size int) *sync.Pool {
	for {
		if p, ok := bufioWriters.Load(size); ok {
			return p.(*sync.Pool)
		}
		if atomic.CompareAndSwapInt32(&assignBufioWriter, 0, 1) {
			var pool = &sync.Pool{New: func() interface{} {
				return bufio.NewWriterSize(nil, size)
			}}
			bufioWriters.Store(size, pool)
			atomic.StoreInt32(&assignBufioWriter, 0)
			return pool
		}
	}
}

// NewBufioReader returns a new bufio.Reader with r.
func NewBufioReader(r io.Reader) *bufio.Reader {
	if v := bufioReaderPool.Get(); v != nil {
		br := v.(*bufio.Reader)
		br.Reset(r)
		return br
	}
	// Note: if this reader size is ever changed, update
	// TestHandlerBodyClose's assumptions.
	return bufio.NewReader(r)
}

// FreeBufioReader frees the bufio.Reader.
func FreeBufioReader(br *bufio.Reader) {
	br.Reset(nil)
	bufioReaderPool.Put(br)
}

// NewBufioWriter returns a new bufio.Writer with w.
func NewBufioWriter(w io.Writer) *bufio.Writer {
	return NewBufioWriterSize(w, bufferBeforeChunkingSize)
}

// NewBufioWriterSize returns a new bufio.Writer with w and size.
func NewBufioWriterSize(w io.Writer, size int) *bufio.Writer {
	pool := assignBufioWriterPool(size)
	bw := pool.Get().(*bufio.Writer)
	bw.Reset(w)
	return bw
}

// FreeBufioWriter frees the bufio.Writer.
func FreeBufioWriter(bw *bufio.Writer) {
	bw.Reset(nil)
	assignBufioWriterPool(bw.Available()).Put(bw)
}

var responsePool = sync.Pool{
	New: func() interface{} {
		return &Response{}
	},
}

var headerPool = sync.Pool{
	New: func() interface{} {
		return make(http.Header)
	},
}

// FreeResponse frees the response.
func FreeResponse(w http.ResponseWriter) {
	if w == nil {
		return
	}
	if res, ok := w.(*Response); ok {
		*res = Response{}
		responsePool.Put(res)
	}
}

func freeHeader(h http.Header) {
	if h == nil {
		return
	}
	for key := range h {
		h.Del(key)
	}
	headerPool.Put(h)
}

// Response implements the http.ResponseWriter interface.
type Response struct {
	req           *http.Request
	conn          net.Conn
	wroteHeader   bool
	rw            *bufio.ReadWriter
	buffer        []byte
	cw            chunkWriter
	handlerHeader http.Header
	setHeader     header
	written       int64 // number of bytes written in body
	noCache       bool
	contentLength int64 // explicitly-declared Content-Length; or -1
	status        int
	hijacked      atomicBool
	dateBuf       [len(TimeFormat)]byte
	clenBuf       [10]byte
	statusBuf     [3]byte

	bufferPool  *sync.Pool
	handlerDone atomicBool // set true when the handler exits
}

type atomicBool int32

func (b *atomicBool) isSet() bool   { return atomic.LoadInt32((*int32)(b)) != 0 }
func (b *atomicBool) setTrue() bool { return atomic.CompareAndSwapInt32((*int32)(b), 0, 1) }

// NewResponse returns a new response.
func NewResponse(req *http.Request, conn net.Conn, rw *bufio.ReadWriter) *Response {
	return NewResponseSize(req, conn, rw, bufferBeforeChunkingSize)
}

// NewResponseSize returns a new response whose buffer has at least the specified
// size.
func NewResponseSize(req *http.Request, conn net.Conn, rw *bufio.ReadWriter, size int) *Response {
	if rw == nil {
		rw = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	}
	bufferPool := assignBufferPool(size)
	res := responsePool.Get().(*Response)
	res.handlerHeader = headerPool.Get().(http.Header)
	res.contentLength = -1
	res.req = req
	res.conn = conn
	res.rw = rw
	res.cw.res = res
	res.bufferPool = bufferPool
	res.buffer = bufferPool.Get().([]byte)
	return res
}

// Header returns the header map that will be sent by
// WriteHeader.
func (w *Response) Header() http.Header {
	return w.handlerHeader
}

// Write writes the data to the connection as part of an HTTP reply.
func (w *Response) Write(data []byte) (n int, err error) {
	if w.hijacked.isSet() {
		return 0, http.ErrHijacked
	}
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	lenData := len(data)
	if lenData == 0 {
		return 0, nil
	}
	if !w.bodyAllowed() {
		return 0, http.ErrBodyNotAllowed
	}
	if !w.cw.chunking {
		offset := w.written
		w.written += int64(lenData) // ignoring errors, for errorKludge
		if w.contentLength != -1 && w.written > w.contentLength {
			return 0, http.ErrContentLength
		}
		if !w.noCache && w.written <= int64(len(w.buffer)) {
			n = copy(w.buffer[offset:w.written], data)
			return
		}
		if !w.noCache {
			w.noCache = true
			if offset > 0 {
				w.cw.Write(w.buffer[:offset])
			}
		}
	}
	return w.cw.Write(data)
}

// WriteHeader sends an HTTP response header with the provided
// status code.
func (w *Response) WriteHeader(code int) {
	if w.hijacked.isSet() {
		return
	}
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	checkWriteHeaderCode(code)
	w.status = code
	if cl := w.handlerHeader.Get(contentLength); cl != emptyString {
		v, err := strconv.ParseInt(cl, 10, 64)
		if err == nil && v >= 0 {
			w.contentLength = v
			w.setHeader.contentLength = cl
		} else {
			w.handlerHeader.Del(contentLength)
		}
	} else if te := w.handlerHeader.Get(transferEncoding); te != emptyString {
		w.setHeader.transferEncoding = te
		if strings.Contains(te, chunked) {
			w.cw.chunking = true
		}
	}
}

// Hijack implements the http.Hijacker interface.
//
// Hijack lets the caller take over the connection.
// After a call to Hijack the HTTP server library
// will not do anything else with the connection.
func (w *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.wroteHeader {
		w.FinishRequest()
	}
	if !w.hijacked.setTrue() {
		return nil, nil, http.ErrHijacked
	}
	return w.conn, w.rw, nil
}

// Flush implements the http.Flusher interface.
//
// Flush writes any buffered data to the underlying connection.
func (w *Response) Flush() {
	if w.hijacked.isSet() {
		return
	}
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if !w.noCache {
		if w.written > 0 {
			w.cw.Write(w.buffer[:w.written])
			w.written = 0
		}
	}
	w.cw.flush()
}

// FinishRequest finishes a request.
func (w *Response) FinishRequest() {
	if !w.handlerDone.setTrue() {
		return
	}
	w.Flush()
	w.cw.close()
	w.rw.Flush()
	// Close the body (regardless of w.closeAfterReply) so we can
	// re-use its bufio.Reader later safely.
	w.req.Body.Close()

	if w.req.MultipartForm != nil {
		w.req.MultipartForm.RemoveAll()
	}
	freeHeader(w.handlerHeader)
	w.handlerHeader = nil
	w.buffer = w.buffer[:cap(w.buffer)]
	w.bufferPool.Put(w.buffer)
	w.buffer = nil
}

// bodyAllowed reports whether a Write is allowed for this response type.
// It's illegal to call this before the header has been flushed.
func (w *Response) bodyAllowed() bool {
	if !w.wroteHeader {
		panic("")
	}
	return bodyAllowedForStatus(w.status)
}

// bodyAllowedForStatus reports whether a given response status code
// permits a body. See RFC 7230, section 3.3.
func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
		return false
	case status == 204:
		return false
	case status == 304:
		return false
	}
	return true
}

func checkWriteHeaderCode(code int) {
	// Issue 22880: require valid WriteHeader status codes.
	// For now we only enforce that it's three digits.
	// In the future we might block things over 599 (600 and above aren't defined
	// at https://httpwg.org/specs/rfc7231.html#status.codes)
	// and we might block under 200 (once we have more mature 1xx support).
	// But for now any three digits.
	//
	// We used to send "HTTP/1.1 000 0" on the wire in responses but there's
	// no equivalent bogus thing we can realistically send in HTTP/2,
	// so we'll consistently panic instead and help people find their bugs
	// early. (We can't return an error from WriteHeader even if we wanted to.)
	if code < 100 || code > 999 {
		panic(fmt.Sprintf("invalid WriteHeader code %v", code))
	}
}

const commaSpaceChunked = ", chunked"

// This should be >= 512 bytes for DetectContentType,
// but otherwise it's somewhat arbitrary.
const bufferBeforeChunkingSize = 2048

// chunkWriter writes to a response's conn buffer, and is the writer
// wrapped by the response.bufw buffered writer.
//
// chunkWriter also is responsible for finalizing the Header, including
// conditionally setting the Content-Type and setting a Content-Length
// in cases where the handler's final output is smaller than the buffer
// size. It also conditionally adds chunk headers, when in chunking mode.
//
// See the comment above (*response).Write for the entire write flow.
type chunkWriter struct {
	res *Response

	// wroteHeader tells whether the header's been written to "the
	// wire" (or rather: w.conn.buf). this is unlike
	// (*response).wroteHeader, which tells only whether it was
	// logically written.
	wroteHeader bool

	// set by the writeHeader method:
	chunking bool // using chunked transfer encoding for reply body
}

func (cw *chunkWriter) Write(p []byte) (n int, err error) {
	if !cw.wroteHeader {
		cw.writeHeader(p)
	}
	if cw.res.req.Method == head {
		// Eat writes.
		return len(p), nil
	}
	if cw.chunking {
		_, err = fmt.Fprintf(cw.res.rw, chunk, len(p))
		if err != nil {
			cw.res.conn.Close()
			return
		}
	}
	n, err = cw.res.rw.Write(p)
	if cw.chunking && err == nil {
		_, err = cw.res.rw.Write(crlf)
	}
	if err != nil {
		cw.res.conn.Close()
	}
	return
}

func (cw *chunkWriter) flush() {
	if !cw.wroteHeader {
		cw.writeHeader(nil)
	}
	cw.res.rw.Flush()
}

func (cw *chunkWriter) close() {
	if !cw.wroteHeader {
		cw.writeHeader(nil)
	}
	if cw.chunking {
		bw := cw.res.rw // conn's bufio writer
		// zero chunk to mark EOF
		bw.Write(zerocrlf)
		//if trailers := cw.res.finalTrailers(); trailers != nil {
		//	trailers.Write(bw) // the writer handles noting errors
		//}
		// final blank line after the trailers (whether
		// present or not)
		bw.Write(crlf)
	}
}

func (cw *chunkWriter) writeHeader(p []byte) {
	if cw.wroteHeader {
		return
	}
	cw.wroteHeader = true
	var w = cw.res
	isHEAD := w.req.Method == "HEAD"

	w.setHeader.date = appendTime(cw.res.dateBuf[:0], time.Now())
	if len(w.setHeader.contentLength) > 0 {
		cw.chunking = false
	} else if cw.chunking {
	} else if w.noCache {
		cw.chunking = true
		if len(w.setHeader.transferEncoding) > 0 {
			if !strings.Contains(w.setHeader.transferEncoding, chunked) {
				w.setHeader.transferEncoding += commaSpaceChunked
			}
		} else {
			w.setHeader.transferEncoding = chunked
		}
	} else if w.handlerDone.isSet() && bodyAllowedForStatus(w.status) && w.handlerHeader.Get(contentLength) == "" && (!w.noCache || !isHEAD || len(p) > 0) {
		w.contentLength = int64(len(p))
		var clen = strconv.AppendInt(w.clenBuf[:0], int64(len(p)), 10)
		w.setHeader.contentLength = *(*string)(unsafe.Pointer(&clen))
	}
	if ct := w.handlerHeader.Get(contentType); ct != emptyString {
		w.setHeader.contentType = ct
	} else {
		if !cw.chunking && len(p) > 0 {
			w.setHeader.contentType = http.DetectContentType(p)
		}
	}
	if co := w.handlerHeader.Get(connection); co != emptyString {
		w.setHeader.connection = co
	}
	w.rw.WriteString(httpVersion)
	if text := http.StatusText(w.status); len(text) > 0 {
		w.rw.Write(strconv.AppendInt(w.statusBuf[:0], int64(w.status), 10))
		w.rw.WriteByte(' ')
		w.rw.WriteString(text)
		w.rw.Write(crlf)
	} else {
		// don't worry about performance
		fmt.Fprintf(w.rw, "%03d status code %d\r\n", w.status, w.status)
	}
	w.setHeader.Write(w.rw.Writer)
	for key := range w.handlerHeader {
		value := w.handlerHeader.Get(key)
		if key == date || key == contentLength || key == transferEncoding || key == contentType || key == connection {
			continue
		}
		if len(key) > 0 && len(value) > 0 {
			w.rw.WriteString(key)
			w.rw.Write(colonSpace)
			w.rw.WriteString(value)
			w.rw.Write(crlf)
		}
	}
	w.rw.Write(crlf)
}

// TimeFormat is the time format to use when generating times in HTTP
// headers. It is like time.RFC1123 but hard-codes GMT as the time
// zone. The time being formatted must be in UTC for Format to
// generate the correct format.
//
// For parsing this time format, see ParseTime.
const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

// appendTime is a non-allocating version of []byte(t.UTC().Format(TimeFormat))
func appendTime(b []byte, t time.Time) []byte {
	const days = "SunMonTueWedThuFriSat"
	const months = "JanFebMarAprMayJunJulAugSepOctNovDec"

	t = t.UTC()
	yy, mm, dd := t.Date()
	hh, mn, ss := t.Clock()
	day := days[3*t.Weekday():]
	mon := months[3*(mm-1):]

	return append(b,
		day[0], day[1], day[2], ',', ' ',
		byte('0'+dd/10), byte('0'+dd%10), ' ',
		mon[0], mon[1], mon[2], ' ',
		byte('0'+yy/1000), byte('0'+(yy/100)%10), byte('0'+(yy/10)%10), byte('0'+yy%10), ' ',
		byte('0'+hh/10), byte('0'+hh%10), ':',
		byte('0'+mn/10), byte('0'+mn%10), ':',
		byte('0'+ss/10), byte('0'+ss%10), ' ',
		'G', 'M', 'T')
}

type header struct {
	date             []byte
	contentLength    string
	contentType      string
	connection       string
	transferEncoding string
}

// Sorted the same as Header.Write's loop.
var headerKeys = [][]byte{
	[]byte("Content-Length"),
	[]byte("Content-Type"),
	[]byte("Connection"),
	[]byte("Transfer-Encoding"),
}
var (
	headerDate = []byte("Date: ")
)
var (
	zerocrlf   = []byte("0\r\n")
	crlf       = []byte("\r\n")
	colonSpace = []byte(": ")
)

// Write writes the headers described in h to w.
//
// This method has a value receiver, despite the somewhat large size
// of h, because it prevents an allocation. The escape analysis isn't
// smart enough to realize this function doesn't mutate h.
func (h header) Write(w *bufio.Writer) {
	if h.date != nil {
		w.Write(headerDate)
		w.Write(h.date)
		w.Write(crlf)
	}
	for i, v := range []string{h.contentLength, h.contentType, h.connection, h.transferEncoding} {
		if len(v) > 0 {
			w.Write(headerKeys[i])
			w.Write(colonSpace)
			w.WriteString(v)
			w.Write(crlf)
		}
	}
}
