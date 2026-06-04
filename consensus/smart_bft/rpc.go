package smart_bft

import (
	"context"
	"fmt"
	"os"

	"github.com/hyperledger-labs/SmartBFT/smartbftprotos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RpcClient struct {
	SmartBftServiceClient
	closer func() error
}

func NewRpcClient(host string, port uint16) (*RpcClient, error) {
	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:%d", host, port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	client := NewSmartBftServiceClient(conn)

	return &RpcClient{
		SmartBftServiceClient: client,
		closer:                conn.Close,
	}, nil
}

func (c *RpcClient) Close() error {
	return c.closer()
}

type RpcServer struct {
	UnimplementedSmartBftServiceServer
	addr  string
	port  uint16
	close chan os.Signal
}

func NewRpcServer(host string, port uint16) (*RpcServer, error) {
	return &RpcServer{
		addr:  host,
		port:  port,
		close: make(chan os.Signal, 2),
	}, nil
}

func (s *RpcServer) HandleMessage(ctx context.Context, in *smartbftprotos.Message) (*Result, error) {
	panic("unimplemented")
}

func (s *RpcServer) FwdMessageReceive(ctx context.Context, in *FwdMessage) (*Result, error) {
	panic("unimplemented")
}
