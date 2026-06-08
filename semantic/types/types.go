package types

import "time"

type Result interface {
	Digest() []byte
	ToBytes() []byte
	Data() interface{}
}

type Config struct {
}

type ImageSemanticConfig struct {
	RpcHost      string        `json:"rpc_host"`
	RpcPort      uint16        `json:"rpc_port"`
	TimeInterval time.Duration `json:"time_interval"`
	Source       string        `json:"source"`
}
