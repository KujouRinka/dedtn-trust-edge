package smart_bft

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"os"
	"strconv"

	"github.com/hyperledger-labs/SmartBFT/smartbftprotos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	rpcpeer "google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
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
	addr string
	port uint16

	parent *Node
	close  chan os.Signal
}

func NewRpcServer(host string, port uint16, parent *Node) (*RpcServer, error) {
	return &RpcServer{
		addr:   host,
		port:   port,
		parent: parent,
		close:  make(chan os.Signal, 2),
	}, nil
}

func (s *RpcServer) HandleMessage(ctx context.Context, in *smartbftprotos.Message) (*emptypb.Empty, error) {
	// todo: SECURITY ISSUE: should deduct sender id from context
	_, ok := rpcpeer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("cannot get peer context")
	}

	senderId, err := strconv.ParseUint(metadata.ValueFromIncomingContext(ctx, "senderId")[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cannot parse sender id: %v", err)
	}
	s.parent.HandleMessage(senderId, in)
	return nil, nil
}

func (s *RpcServer) ReqMessageCall(ctx context.Context, in *RequestEnvelope) (*emptypb.Empty, error) {
	b, err := proto.Marshal(in)
	if err != nil {
		return nil, err
	}
	return nil, s.parent.SubmitRequest(b)
}

func (s *RpcServer) PullLedger(ctx context.Context, in *emptypb.Empty) (*LedgerBytes, error) {
	s.parent.ledger.mu.Lock()
	defer s.parent.ledger.mu.Unlock()

	b, err := s.parent.ledger.toBytes()
	if err != nil {
		return nil, err
	}
	return &LedgerBytes{RawData: b}, nil
}

func (s *RpcServer) PullLatestMetadata(ctx context.Context, in *emptypb.Empty) (*Metadata, error) {
	s.parent.ledger.mu.Lock()
	defer s.parent.ledger.mu.Unlock()

	md := &smartbftprotos.ViewMetadata{}
	err := proto.Unmarshal(s.parent.ledger.Decision.Proposal.Metadata, md)
	if err != nil {
		s.parent.logger.Panic("should not return error")
	}
	return &Metadata{
		ViewId:          md.ViewId,
		DecisionsInView: md.DecisionsInView,
		LatestSequence:  md.LatestSequence,
	}, nil
}

type peer struct {
	*RpcClient
	id        uint64
	host      string
	port      uint16
	publicKey *ecdsa.PublicKey
}

func newPeer(id uint64, host string, port uint16, pubKey *ecdsa.PublicKey) (*peer, error) {
	return &peer{
		id:        id,
		host:      host,
		port:      port,
		publicKey: pubKey,
	}, nil
}

func (p *peer) Id() uint64 {
	return p.id
}

func (p *peer) Connect() error {
	client, err := NewRpcClient(p.host, p.port)
	if err != nil {
		return err
	}
	p.RpcClient = client
	return nil
}
