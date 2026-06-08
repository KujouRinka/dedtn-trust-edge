package yolo

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/kujourinka/dedtn-trust-edge/semantic/device"
	"github.com/kujourinka/dedtn-trust-edge/semantic/types"
)

type Semantic struct {
	Reply *DetectReply
}

func (s *Semantic) Digest() []byte {
	panic("unimplemented")
}

func (s *Semantic) ToBytes() []byte {
	panic("unimplemented")
}

func (s *Semantic) Data() interface{} {
	return s.Reply
}

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

func (c *RpcClient) Semanticize(ctx context.Context, data device.Data) (types.Result, error) {
	resp, err := c.DetectFrame(ctx, &FrameRequest{
		ImageData: data.Data(),
		CameraId:  0,
	})
	if err != nil {
		return nil, err
	}
	return &Semantic{Reply: resp}, nil
}
