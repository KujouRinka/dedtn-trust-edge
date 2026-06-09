package yolo

import (
	semantictypes "github.com/kujourinka/dedtn-trust-edge/semantic/types"
)

type Verifier struct {
}

var _ semantictypes.Verifier = (*Verifier)(nil)

func NewVerifier() (*Verifier, error) {
	// todo: implemented
	return &Verifier{}, nil
}

func (v *Verifier) VerifySemantic(result semantictypes.Result) (bool, error) {
	// todo: unimplemented
	return true, nil
}
