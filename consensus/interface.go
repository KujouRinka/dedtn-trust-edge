package consensus

import (
	"github.com/kujourinka/dedtn-trust-edge/semantic"
)

type Node interface {
	Run() error
	Stop() error

	SubmitSemantic(msg semantic.Data) error

	// // SendConsensus receives SmartBFT message from other replica
	// SendConsensus() error
	// // SendTransaction receives forwarded client request from other node
	// SendTransaction() error
}
