package smart_bft

import (
	"bytes"
	"encoding/gob"
	"sync"

	bfttypes "github.com/hyperledger-labs/SmartBFT/pkg/types"
)

type ledger struct {
	mu sync.Mutex
	ledgerWrapper
}

type ledgerWrapper struct {
	// statistic data
	SemanticCount uint64
	VoteCount     uint64

	Decision bfttypes.Decision
	Sequence uint64

	Submitted map[string]struct{}
	// map[SemanticId][]*SemanticVote
	CommittedVote map[string][]*SemanticVote
}

func ledgerWrapperFromBytes(b []byte) (*ledgerWrapper, error) {
	var wrapper ledgerWrapper
	err := gob.NewDecoder(bytes.NewReader(b)).Decode(&wrapper)
	if err != nil {
		return nil, err
	}
	return &wrapper, nil
}

func (l *ledger) toBytes() ([]byte, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var buffer bytes.Buffer
	err := gob.NewEncoder(&buffer).Encode(l.ledgerWrapper)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func init() {
	gob.Register(ledgerWrapper{})
}
