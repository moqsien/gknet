package conn

import "sync"

var readBufferPool = sync.Pool{}

func (that *Conn) GetBufferFromPool() []byte {
	if readBufferPool.New == nil {
		readBufferPool.New = func() any {
			return make([]byte, that.Poller.ReadBufferSize)
		}
	}
	return readBufferPool.Get().([]byte)
}

func (that *Conn) PutBufferToPool(buf []byte) {
	readBufferPool.Put(buf)
}
