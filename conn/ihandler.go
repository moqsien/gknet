package conn

type IEventHandler interface {
	OnAccept(*Conn) error
	OnOpen(*Context) (data []byte, err error)
	OnTrack(*Context) error
	OnClose(*Context) error
}
