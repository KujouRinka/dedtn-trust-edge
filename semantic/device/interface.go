package device

import "io"

type Camera interface {
	io.Closer
	CurrentFrame() ([]byte, error)
}
