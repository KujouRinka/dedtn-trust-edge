package semantic

import "time"

type Data interface {
	Digest() []byte
	ToBytes() []byte
}

type SemanticClaim struct {
	ClientId  string
	RequestId string
	Payload   interface{}
}

func (s *SemanticClaim) Digest() []byte {
	return nil
}

func (s *SemanticClaim) ToBytes() []byte {
	return nil
}

type Config struct {
}

type ImageSemanticConfig struct {
	RpcHost      string        `json:"rpc_host"`
	RpcPort      uint16        `json:"rpc_port"`
	TimeInterval time.Duration `json:"time_interval"`
	Source       string        `json:"source"`
}
