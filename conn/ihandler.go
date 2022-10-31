package conn

type IEventHandler interface {
	OnOpen(*Conn) (data []byte, err error)
	OnTrack(*Conn) error
	OnAccept(*Conn) error
	OnClose(*Conn) error
}
