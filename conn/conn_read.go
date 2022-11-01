package conn

import (
	"io"
)

func (that *Conn) Read(p []byte) (n int, err error) {
	if that.InBuffer.IsEmpty() {
		n = copy(p, that.Buffer)
		that.Buffer = that.Buffer[n:]
		if n == 0 && len(p) > 0 {
			err = io.EOF
		}
		return
	}
	n, _ = that.InBuffer.Read(p)
	if n == len(p) {
		return
	}
	m := copy(p[n:], that.Buffer)
	n += m
	that.Buffer = that.Buffer[m:]
	return
}
