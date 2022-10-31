package conn

type AsyncCallback func(c *Conn) error

type AsyncWriteHook struct {
	Go   AsyncCallback
	Data []byte
}

type AsyncWritevHook struct {
	Go   AsyncCallback
	Data [][]byte
}
