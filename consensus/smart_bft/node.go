package smart_bft

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/google/uuid"
	smartbft "github.com/hyperledger-labs/SmartBFT/pkg/consensus"
	bfttypes "github.com/hyperledger-labs/SmartBFT/pkg/types"
	"github.com/hyperledger-labs/SmartBFT/smartbftprotos"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	"github.com/kujourinka/dedtn-trust-edge/consensus"
	"github.com/kujourinka/dedtn-trust-edge/logger"
	"github.com/kujourinka/dedtn-trust-edge/semantic"
	"github.com/kujourinka/dedtn-trust-edge/semantic/yolo"
)

type Node struct {
	peers  map[uint64]*peer
	server *RpcServer

	consensus *smartbft.Consensus
	sfNode    *snowflake.Node

	key *ecdsa.PrivateKey

	clock       *time.Ticker
	secondClock *time.Ticker
	ctx         context.Context

	idName string
	// committed data field
	deliverCount atomic.Uint64
}

func NewSmartPBFServer(config *Config) (consensus.Node, error) {
	server, err := NewRpcServer(config.ListenAddr, config.ListenPort, nil)
	if err != nil {
		return nil, err
	}

	peers := make(map[uint64]*peer)
	for _, p := range config.Peers {
		np, err := newPeer(p.Id, p.Addr, p.Port, nil)
		if err != nil {
			return nil, err
		}
		_, ok := peers[p.Id]
		if ok {
			return nil, fmt.Errorf("peer id repeated: %d", p.Id)
		}
		peers[p.Id] = np
	}

	sfNode, err := snowflake.NewNode(int64(config.Id))
	node := &Node{
		peers:       peers,
		server:      server,
		sfNode:      sfNode,
		clock:       time.NewTicker(time.Second),
		secondClock: time.NewTicker(time.Second),
		ctx:         context.Background(),

		idName: "node" + strconv.FormatUint(config.Id, 10),
	}
	server.parent = node

	// met := &disabled.Provider{}
	// walMet := wal.NewMetrics(met, "label1")

	// nodeDir := filepath.Join(config.NodeDir, fmt.Sprintf("node%d", config.Id))
	// writeAheadLog, err := wal.Create(logger.Logger, nodeDir, &wal.Options{Metrics: walMet.With("label1", "val1")})
	// if err != nil {
	// 	return nil, err
	// }

	consensusCfg := bfttypes.DefaultConfig
	consensusCfg.SelfID = config.Id
	node.consensus = &smartbft.Consensus{
		Config:      consensusCfg,
		Application: node,
		Assembler:   node,
		// todo: WAL
		WAL: &MemoryWAL{},
		// WALInitialContent:  nil,
		Comm:               node,
		Signer:             node,
		Verifier:           node,
		MembershipNotifier: node,
		RequestInspector:   node,
		Synchronizer:       node,
		Logger:             logger.Logger,
		// Metrics:            nil,
		Metadata: &smartbftprotos.ViewMetadata{
			LatestSequence: 0,
			ViewId:         0,
		},
		// LastProposal:       bfttypes.Proposal{},
		// LastSignatures:     nil,
		Scheduler:         node.secondClock.C,
		ViewChangerTicker: node.clock.C,
		// Pool:               nil,
	}
	return node, nil
}

func (s *Node) Run() error {
	if err := s.consensus.Start(); err != nil {
		return err
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.server.addr, s.server.port))
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	RegisterSmartBftServiceServer(grpcServer, s.server)
	go func() {
		logger.Logger.Info("grpc server started", zap.Uint64("NodeId", s.consensus.Config.SelfID))
		if err := grpcServer.Serve(listener); err != nil {
			logger.Logger.Warnf("grpc server cannot start: %v", err)
		}
	}()

	signal.Notify(
		s.server.close,
		os.Interrupt,
		syscall.SIGTERM,
	)
	<-s.server.close

	return nil
}

func (s *Node) SubmitSemantic(msg semantic.Data) error {
	m, ok := msg.Data().(*yolo.DetectReply)
	if !ok {
		return fmt.Errorf("unsupported message type")
	}

	var envelope RequestEnvelope
	envelope.SubmitterId = s.idName
	envelope.ClientId = s.idName
	// envelope.RequestId = s.sfNode.Generate().String()
	envelope.RequestId = uuid.New().String()
	// todo: signature
	envelope.SubmitterSignature = nil
	envelope.Payload = &RequestEnvelope_SemanticBox{&SemanticBox{Reply: m}}

	b, err := proto.Marshal(&envelope)
	if err != nil {
		return err
	}
	return s.SubmitRequest(b)
}

func (s *Node) Stop() error {
	var errs []error
	for _, p := range s.peers {
		if err := p.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Synchronizer interface

func (s *Node) Sync() bfttypes.SyncResponse {
	panic("unimplemented")
}

// RequestInspector interface

func (s *Node) RequestID(req []byte) bfttypes.RequestInfo {
	// Should return client id and request id
	var envelope RequestEnvelope
	if err := proto.Unmarshal(req, &envelope); err != nil {
		logger.Logger.Panic("should not return error", zap.Error(err))
	}

	return bfttypes.RequestInfo{ClientID: envelope.ClientId, ID: envelope.RequestId}
}

// Application interface

// Deliver delivers the given proposal and signatures.
// After the call returns we assume that this proposal is stored in persistent memory.
// It returns whether this proposal was a reconfiguration and the current config.
func (s *Node) Deliver(proposal bfttypes.Proposal, signature []bfttypes.Signature) bfttypes.Reconfig {
	// if is trustable message then do on-chain
	// if is fake message, we should call a smart contract to recalculate trust score
	// Should we do these in a goroutine?
	var blockRequest BlockRequest
	if err := proto.Unmarshal(proposal.Payload, &blockRequest); err != nil {
		// todo:
		panic("panic")
	}
	for _, envelope := range blockRequest.Requests {
		s.deliverCount.Add(1)

		switch envelope.Payload.(type) {
		case *RequestEnvelope_SemanticBox:
		case *RequestEnvelope_SemanticVote:
		case *RequestEnvelope_SemanticDecision:
		default:
			panic("panic")
		}
	}

	return bfttypes.Reconfig{InLatestDecision: false}
}

// Assembler interface

func (s *Node) AssembleProposal(metadata []byte, requests [][]byte) bfttypes.Proposal {
	// assemble proposal from other node to bfttype.Proposal
	// only primary be called (?)
	// get message from SendTransaction (?)
	// var blockRequest BlockRequest
	// todo: performance
	// payload := encodeBlockDataRaw(requests)

	var blockRequest BlockRequest
	for _, req := range requests {
		var envelope RequestEnvelope
		if err := proto.Unmarshal(req, &envelope); err != nil {
			logger.Logger.Panic("should not return error", zap.Error(err))
		}
		// do something with envelope
		blockRequest.Requests = append(blockRequest.Requests, &envelope)
	}
	payload, err := proto.Marshal(&blockRequest)
	if err != nil {
		logger.Logger.Panic("should not return error", zap.Error(err))
	}

	return bfttypes.Proposal{
		Payload:  payload,
		Header:   nil,
		Metadata: metadata,
		// todo:
		VerificationSequence: 0,
	}
}

// Comm interface
func (s *Node) SendConsensus(targetID uint64, m *smartbftprotos.Message) {
	// send SmartBFT protocol message, such as prepare, commit
	// send this message to other node
	// other nodes received this message should call consensus.HandleMesssage()

	// todo: SECURITY
	ctx := metadata.AppendToOutgoingContext(s.ctx, "senderId", strconv.FormatUint(s.consensus.Config.SelfID, 10))
	_, err := s.peers[targetID].HandleMessage(ctx, m)
	if err != nil {
		logger.Logger.Error("rpc HandleMessage error", zap.Error(err))
	}
}

func (s *Node) SendTransaction(targetID uint64, request []byte) {
	// forward client request to other node, usually primary node
	// todo: performance
	var envelope RequestEnvelope
	if err := proto.Unmarshal(request, &envelope); err != nil {
		logger.Logger.Panic("should not return error", zap.Error(err))
	}
	_, err := s.peers[targetID].ReqMessageCall(s.ctx, &envelope)
	if err != nil {
		logger.Logger.Error("rpc SendTransaction error", zap.Error(err))
	}
}

func (s *Node) Nodes() []uint64 {
	ids := make([]uint64, 0, len(s.peers))
	for id := range s.peers {
		ids = append(ids, id)
	}
	ids = append(ids, s.consensus.Config.SelfID)
	return ids
}

// Signer
func (s *Node) Sign(msg []byte) []byte {
	// todo: sig
	return nil
}

func (s *Node) SignProposal(proposal bfttypes.Proposal, auxiliaryInput []byte) *bfttypes.Signature {
	// correspond to `VerifyConsenterSig`
	// todo: sig
	// proposal.Digest()

	// `Msg`: original message (?)
	return &bfttypes.Signature{
		ID:    s.consensus.Config.SelfID,
		Value: nil,
		Msg:   nil,
	}
}

// Verifier interface

// VerifyProposal verifies the given proposal and returns the included requests' info.
func (s *Node) VerifyProposal(proposal bfttypes.Proposal) ([]bfttypes.RequestInfo, error) {
	infos := s.RequestsFromProposal(proposal)
	return infos, nil
}

// VerifyRequest verifies the given request and returns its info.
// return err != nil if this request is invalid
func (s *Node) VerifyRequest(val []byte) (bfttypes.RequestInfo, error) {
	var envelope RequestEnvelope
	if err := proto.Unmarshal(val, &envelope); err != nil {
		return bfttypes.RequestInfo{}, fmt.Errorf("cannot decode message: %v", err)
	}
	return bfttypes.RequestInfo{
		ClientID: envelope.ClientId,
		ID:       envelope.RequestId,
	}, nil
}

// VerifyConsenterSig verifies the signature for the given proposal.
// It returns the auxiliary data in the signature.
func (s *Node) VerifyConsenterSig(signature bfttypes.Signature, prop bfttypes.Proposal) ([]byte, error) {
	return nil, nil
}

// VerifySignature verifies the signature.
func (s *Node) VerifySignature(signature bfttypes.Signature) error {
	// todo: verify `signature.Value`
	return nil
}

// VerificationSequence returns the current verification sequence.
func (s *Node) VerificationSequence() uint64 {
	// todo:
	return 0
}

// RequestsFromProposal returns from the given proposal the included requests' info
func (s *Node) RequestsFromProposal(proposal bfttypes.Proposal) []bfttypes.RequestInfo {
	var blockRequest BlockRequest
	if err := proto.Unmarshal(proposal.Payload, &blockRequest); err != nil {
		logger.Logger.Panic("should not return error", zap.Error(err))
	}
	infos := make([]bfttypes.RequestInfo, 0, len(blockRequest.Requests))
	for _, r := range blockRequest.Requests {
		info := bfttypes.RequestInfo{
			ClientID: r.ClientId,
			ID:       r.RequestId,
		}
		infos = append(infos, info)
	}
	return infos
}

// AuxiliaryData extracts the auxiliary data from a signature's message
func (s *Node) AuxiliaryData([]byte) []byte {
	return nil
}

// MembershipNotifier interface

func (s *Node) MembershipChange() bool {
	return false
}

// Other
func (s *Node) HandleMessage(sender uint64, m *smartbftprotos.Message) {
	s.consensus.HandleMessage(sender, m)
}

func (s *Node) HandleRequest(sender uint64, req []byte) {
	s.consensus.HandleRequest(sender, req)
}

func (s *Node) SubmitRequest(req []byte) error {
	return s.consensus.SubmitRequest(req)
}

type MemoryWAL struct {
	mu      sync.Mutex
	entries [][]byte
}

func (w *MemoryWAL) Append(entry []byte, truncateTo bool) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	copied := append([]byte(nil), entry...)
	if truncateTo {
		w.entries = [][]byte{copied}
		return nil
	}

	w.entries = append(w.entries, copied)
	return nil
}
