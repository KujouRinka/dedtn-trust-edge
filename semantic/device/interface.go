package device

import "io"

type Camera interface {
	io.Closer
	CurrentFrame() (Data, error)
}

type Data interface {
	deviceData()
	Data() []byte
}
