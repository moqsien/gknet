package conn

type EventHandler interface {
	OnOpen(*Conn) (data []byte, err error)
	OnTrack(*Conn) error
	OnClose(*Conn, error) error
}
