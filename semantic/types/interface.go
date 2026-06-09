package types

import (
	"context"

	"github.com/kujourinka/dedtn-trust-edge/semantic/device"
)

type Generator interface {
	Run() error
	ReadChan() <-chan Result
}

type Semanticist interface {
	Semanticize(ctx context.Context, data device.Data) (result Result, err error)
	Close() error
}

type Verifier interface {
	VerifySemantic(result Result) (ok bool, err error)
}
