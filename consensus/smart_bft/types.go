package smart_bft

import (
	semantictypes "github.com/kujourinka/dedtn-trust-edge/semantic/types"
	"google.golang.org/protobuf/encoding/protowire"
)

type Config struct {
	Id      uint64
	NodeDir string

	ListenAddr string
	ListenPort uint16

	Peers    []*PeerCfg
	verifier semantictypes.Verifier
}

type PeerCfg struct {
	Id   uint64
	Addr string
	Port uint16
}

type SemanticDecisionRequest struct {
}

func encodeBlockDataRaw(requests [][]byte) []byte {
	var totalLen int
	for _, raw := range requests {
		totalLen += 1 + protowire.SizeVarint(uint64(len(raw))) + len(raw)
	}

	buf := make([]byte, 0, totalLen)
	for _, raw := range requests {
		buf = protowire.AppendTag(buf, 1, protowire.BytesType)
		buf = protowire.AppendBytes(buf, raw)
	}
	return buf
}
