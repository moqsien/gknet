package conn

const (
	IovMax = 1024
)

type AsyncCallback func(c *Conn) error

type AsyncWriteHook struct {
	Go   AsyncCallback
	Data []byte
}

type AsyncWritevHook struct {
	Go   AsyncCallback
	Data [][]byte
}

type EventHandler interface {
	OnOpen(*Conn) (data []byte, err error)
	OnTrack(*Conn) error
}
