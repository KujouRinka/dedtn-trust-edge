package consensus

import (
	semantictypes "github.com/kujourinka/dedtn-trust-edge/semantic/types"
)

type Node interface {
	Run() error
	Stop() error

	SubmitSemantic(msg semantictypes.Result) error

	// // SendConsensus receives SmartBFT message from other replica
	// SendConsensus() error
	// // SendTransaction receives forwarded client request from other node
	// SendTransaction() error
}
