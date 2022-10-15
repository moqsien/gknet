package errs

import "errors"

var (
	ErrAcceptSocket   = errors.New("accept a new connection error")
	ErrEngineShutdown = errors.New("server is going to be shutdown")
	ErrUnsupportedOp  = errors.New("unsupported operation")
)
