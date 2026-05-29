package yolo

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RpcClient struct {
	YoloServiceClient

	closer func() error
}

func (c *RpcClient) Close() error {
	return c.closer()
}

func NewRpcClient(host string, port uint16) (*RpcClient, error) {
	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:%d", host, port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	client := NewYoloServiceClient(conn)

	return &RpcClient{
		YoloServiceClient: client,
		closer:            conn.Close,
	}, nil
}
