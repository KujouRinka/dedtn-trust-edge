package semantic

import "time"

type Data interface {
	Digest() []byte
	ToBytes() []byte
	Data() interface{}
}

type SemanticClaim struct {
	Payload interface{}
}

func (s *SemanticClaim) Digest() []byte {
	return nil
}

func (s *SemanticClaim) ToBytes() []byte {
	return nil
}

func (s *SemanticClaim) Data() interface{} {
	return s.Payload
}

type Config struct {
}

type ImageSemanticConfig struct {
	RpcHost      string        `json:"rpc_host"`
	RpcPort      uint16        `json:"rpc_port"`
	TimeInterval time.Duration `json:"time_interval"`
	Source       string        `json:"source"`
}
