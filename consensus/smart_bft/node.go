package smart_bft

import (
	"cmp"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"reflect"
	"slices"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/google/uuid"
	smartbft "github.com/hyperledger-labs/SmartBFT/pkg/consensus"
	bfttypes "github.com/hyperledger-labs/SmartBFT/pkg/types"
	"github.com/hyperledger-labs/SmartBFT/smartbftprotos"
	"github.com/kujourinka/dedtn-trust-edge/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	"github.com/kujourinka/dedtn-trust-edge/consensus"
	"github.com/kujourinka/dedtn-trust-edge/logger"
	semantictypes "github.com/kujourinka/dedtn-trust-edge/semantic/types"
	"github.com/kujourinka/dedtn-trust-edge/semantic/yolo"
)

type Node struct {
	peers  map[uint64]*peer
	server *RpcServer

	consensus        *smartbft.Consensus
	sfNode           *snowflake.Node
	semanticVerifier semantictypes.Verifier

	key *ecdsa.PrivateKey

	clock          *time.Ticker
	schedulerClock *time.Ticker
	ctx            context.Context
	cancel         context.CancelFunc

	idName string
	logger *logger.SelfLogger

	ledger *ledger
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
	if err != nil {
		panic("fix me")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	node := &Node{
		peers:            peers,
		server:           server,
		sfNode:           sfNode,
		semanticVerifier: config.verifier,
		clock:            time.NewTicker(300 * time.Millisecond),
		schedulerClock:   time.NewTicker(100 * time.Millisecond),
		// ctx:              context.Background(),
		ctx:    ctx,
		cancel: cancel,

		idName: "node" + strconv.FormatUint(config.Id, 10),

		ledger: &ledger{
			ledgerWrapper: ledgerWrapper{
				Submitted:     make(map[string]struct{}),
				CommittedVote: make(map[string][]*SemanticVote),
			},
		},
	}
	node.logger = &logger.SelfLogger{Logger: logger.Logger.Named(node.idName)}
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
	consensusCfg.RequestPoolSize = 20000
	consensusCfg.IncomingMessageBufferSize = 20000
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
		Logger:             node.logger,
		// Metrics:            nil,
		Metadata: &smartbftprotos.ViewMetadata{
			LatestSequence: 0,
			ViewId:         0,
		},
		// LastProposal:       bfttypes.Proposal{},
		// LastSignatures:     nil,
		Scheduler:         node.schedulerClock.C,
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

	for _, p := range s.peers {
		if err := p.Connect(); err != nil {
			return err
		}
	}

	signal.Notify(
		s.server.close,
		os.Interrupt,
		syscall.SIGTERM,
	)
	<-s.server.close
	grpcServer.GracefulStop()

	return nil
}

func (s *Node) SubmitSemantic(msg semantictypes.Result) error {
	envelope := s.newEnvelope()

	switch v := msg.Data().(type) {
	case *yolo.DetectReply:
		envelope.Payload = &RequestEnvelope_SemanticBox{&SemanticBox{Reply: v}}
	default:
		err := fmt.Errorf("attempt to submit unsupported semantic result")
		logger.Logger.Errorf("%s error: %v", s.idName, err)
		return err
	}

	if err := s.signEnvelope(envelope); err != nil {
		logger.Logger.Errorf("sign envelope failed: %v", err)
		return err
	}

	b, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}
	return s.SubmitRequest(b)
}

func (s *Node) SubmitVote(semanticId string, ok bool) error {
	envelope := s.newEnvelope()

	var voteValue VoteValue
	if ok {
		voteValue = VoteValue_VOTE_ACCEPT
	} else {
		voteValue = VoteValue_VOTE_REJECT
	}
	vote := &SemanticVote{
		SemanticId:      semanticId,
		ObservationHash: "",
		Round:           0,
		VoterId:         s.idName,
		Vote:            voteValue,
		// todo: must sign
		VoterSignature: nil,
	}
	// todo: fill vote.VoterSignature

	envelope.Payload = &RequestEnvelope_SemanticVote{SemanticVote: vote}

	if err := s.signEnvelope(envelope); err != nil {
		logger.Logger.Errorf("sign envelope failed: %v", err)
		return err
	}
	b, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}
	return s.SubmitRequest(b)
}

func (s *Node) SubmitDecision(what ...interface{}) error {
	panic("unimplemented")
	return nil
}

func (s *Node) newEnvelope() *RequestEnvelope {
	var envelope RequestEnvelope
	envelope.SubmitterId = s.idName
	envelope.ClientId = s.idName
	// envelope.RequestId = s.sfNode.Generate().String()
	envelope.RequestId = uuid.New().String()
	return &envelope
}

func (s *Node) signEnvelope(envelope *RequestEnvelope) error {
	// todo:
	envelope.SubmitterSignature = nil
	return nil
}

func (s *Node) semanticId(envelope *RequestEnvelope) string {
	// todo:
	return fmt.Sprintf("%s:%s", envelope.ClientId, envelope.RequestId)
}

func (s *Node) Stop() error {
	var errs []error
	for _, p := range s.peers {
		if p.RpcClient == nil {
			continue
		}
		if err := p.Close(); err != nil {
			errs = append(errs, err)
		}
		s.consensus.Stop()
	}
	return errors.Join(errs...)
}

// Synchronizer interface

func (s *Node) Sync() bfttypes.SyncResponse {
	// todo:

	// get metadata random
	var remoteWrapper *ledgerWrapper

	n := len(s.peers)/3 + 1
	metadatas := make([]utils.Pair[*Metadata, uint64], 0, n)
	// for _, i := range utils.RandomIntsButV2(n, 1, len(s.peers)+1, int(s.consensus.Config.SelfID)) {
	for k, p := range s.peers {
		s.logger.Warnf("Sync: current idx: %d", k)
		data, err := p.PullLatestMetadata(s.ctx, nil)
		if err != nil {
			// s.logger.Panicf("fix this: %v", err)
			return bfttypes.SyncResponse{}
		}
		metadatas = append(metadatas, utils.Pair[*Metadata, uint64]{First: data, Second: k})
		n--
		if n == 0 {
			break
		}
	}
	slices.SortFunc(metadatas, func(a, b utils.Pair[*Metadata, uint64]) int {
		return cmp.Compare(b.First.LatestSequence, a.First.LatestSequence)
	})

	var fp utils.Pair[*Metadata, uint64]
	for _, p := range metadatas {
		b, err := s.peers[p.Second].PullLedger(s.ctx, nil)
		if err != nil {
			s.logger.Errorf("Sync: PullLedger failed: %v", err)
			continue
		}
		remoteWrapper, err = ledgerWrapperFromBytes(b.RawData)
		if err != nil {
			s.logger.Errorf("Sync: decode ledger bytes failed: %v", err)
			continue
		}
		fp = p
		break
	}
	if remoteWrapper == nil {
		// panic("fix this")
		return bfttypes.SyncResponse{}
	}

	remoteMd := &smartbftprotos.ViewMetadata{}
	err := proto.Unmarshal(remoteWrapper.Decision.Proposal.Metadata, remoteMd)
	if err != nil {
		s.logger.Panic("should not return error")
	}

	s.ledger.mu.Lock()
	defer s.ledger.mu.Unlock()

	md := &smartbftprotos.ViewMetadata{}
	err = proto.Unmarshal(s.ledger.Decision.Proposal.Metadata, md)
	if err != nil {
		s.logger.Panic("should not return error")
	}

	response := bfttypes.SyncResponse{
		Latest:   s.ledger.Decision,
		Reconfig: bfttypes.ReconfigSync{InReplicatedDecisions: false},
	}

	// compare sequence
	if remoteMd.LatestSequence > md.LatestSequence {
		// update
		s.logger.Warnf("Sync: pull ledger from: %d, data %v", fp.Second, fp.First)
		response.Latest = remoteWrapper.Decision
		s.ledger.Decision = remoteWrapper.Decision
		s.ledger.Submitted = remoteWrapper.Submitted
		s.ledger.CommittedVote = remoteWrapper.CommittedVote
		s.ledger.Sequence = remoteWrapper.Sequence

		// update statistic field
		s.ledger.SemanticCount = remoteWrapper.SemanticCount
		s.ledger.VoteCount = remoteWrapper.VoteCount
	} else {
		response.Latest = s.ledger.Decision
	}

	return response
}

// RequestInspector interface

func (s *Node) RequestID(req []byte) bfttypes.RequestInfo {
	// Should return client id and request id
	var envelope RequestEnvelope
	if err := proto.Unmarshal(req, &envelope); err != nil {
		logger.Logger.Panic("RequestID: should not return error", zap.Error(err))
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

	s.ledger.mu.Lock()
	defer s.ledger.mu.Unlock()

	s.ledger.Decision = bfttypes.Decision{
		Proposal:   proposal,
		Signatures: signature,
	}
	v := &smartbftprotos.ViewMetadata{}
	err := proto.Unmarshal(s.ledger.Decision.Proposal.Metadata, v)
	if err != nil {
		s.logger.Panicf("VerificationSequence: %v", err)
	}
	s.ledger.Sequence = v.LatestSequence
	s.logger.Warnf("Deliver: write sequence: %d", v.LatestSequence)

	for _, envelope := range blockRequest.Requests {

	switchEntry:
		switch v := envelope.Payload.(type) {
		case *RequestEnvelope_SemanticBox:
			s.ledger.SemanticCount += 1

			// create a new goroutine for SemanticVote
			go func(envelope *RequestEnvelope, v *RequestEnvelope_SemanticBox) {
				// logger.Logger.Warnf("%s %v", s.idName, v)
				ok, err := s.semanticVerifier.VerifySemantic(v.SemanticBox.Reply)
				if err != nil {
					// todo:
					s.logger.Errorf("Deliver: verify semantic error: %v", err)
					return
				}
				s.logger.Infof(
					"got response from validator, client id: %v, request id: %v, status: %v",
					envelope.ClientId, envelope.RequestId, ok,
				)

				if err := s.SubmitVote(s.semanticId(envelope), ok); err != nil {
					s.logger.Panicf("submit vote failed: %v", err)
				}
			}(envelope, v)
		case *RequestEnvelope_SemanticVote:
			s.ledger.VoteCount += 1
			// store vote result reached consensus
			semanticId := v.SemanticVote.SemanticId
			logger.Logger.Infof("Deliver: reach vote consensus of semantic id: %s", semanticId)

			votes := s.ledger.CommittedVote[semanticId]
			if votes == nil {
				votes = make([]*SemanticVote, 0, len(s.peers)/3*2+1)
			}
			// todo: should verify signature of SemanticVote here ?

			// check duplicate voter
			for _, vote := range votes {
				if vote.VoterId == v.SemanticVote.VoterId {
					break switchEntry
				}
			}
			s.ledger.CommittedVote[semanticId] = append(votes, v.SemanticVote)
			s.logger.Warnf("Deliver: vote committed: semantic id: %s, vote id: %s", semanticId, v.SemanticVote.Vote)
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
			logger.Logger.Panic("Assemble: should not return error", zap.Error(err))
		}
		// do something with envelope
		blockRequest.Requests = append(blockRequest.Requests, &envelope)
	}
	payload, err := proto.Marshal(&blockRequest)
	if err != nil {
		logger.Logger.Panic("Assemble: should not return error", zap.Error(err))
	}

	md := &smartbftprotos.ViewMetadata{}
	err = proto.Unmarshal(metadata, md)
	if err != nil {
		s.logger.Panicf("cannot decode metadata: %v", err)
	}
	s.logger.Warnf("Assemble: metadata: %v", md)

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
		s.logger.Error("SendConsensus: rpc HandleMessage error", zap.Error(err))
	}
}

func (s *Node) SendTransaction(targetID uint64, request []byte) {
	// forward client request to other node, usually primary node
	// todo: performance
	var envelope RequestEnvelope
	if err := proto.Unmarshal(request, &envelope); err != nil {
		s.logger.Panic("should not return error", zap.Error(err))
	}
	_, err := s.peers[targetID].ReqMessageCall(context.Background(), &envelope)
	if err != nil {
		s.logger.Errorf("SendTransaction: rpc SendTransaction error, msg type: %v, err: %v",
			reflect.ValueOf(envelope.Payload).Type(),
			err,
		)
	}
}

func (s *Node) Nodes() []uint64 {
	ids := make([]uint64, 0, len(s.peers))
	for id := range s.peers {
		ids = append(ids, id)
	}
	ids = append(ids, s.consensus.Config.SelfID)

	// for debug
	if len(ids) != 10 {
		panic("Nodes: peers count should be 10")
	}
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
	var blockRequest BlockRequest
	if err := proto.Unmarshal(proposal.Payload, &blockRequest); err != nil {
		// todo:
		// return nil, fmt.Errorf("cannot decode proposal payload: %v", err)
		logger.Logger.Panic("should not return error", zap.Error(err))
	}
	infos := make([]bfttypes.RequestInfo, 0, len(blockRequest.Requests))
	for _, r := range blockRequest.Requests {
		if r.ClientId == "" {
			// todo:
			// return bfttypes.RequestInfo{}, fmt.Errorf("empty client id")
			s.logger.Panic("VerifyRequest: empty client id")
		}
		if r.RequestId == "" {
			// todo:
			// return bfttypes.RequestInfo{}, fmt.Errorf("empty request id")
			s.logger.Panic("VerifyRequest: empty request id")
		}
		info := bfttypes.RequestInfo{
			ClientID: r.ClientId,
			ID:       r.RequestId,
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// VerifyRequest verifies the given request and returns its info.
// return err != nil if this request is invalid
func (s *Node) VerifyRequest(val []byte) (bfttypes.RequestInfo, error) {
	var envelope RequestEnvelope
	if err := proto.Unmarshal(val, &envelope); err != nil {
		// todo:
		// return bfttypes.RequestInfo{}, fmt.Errorf("cannot decode message: %v", err)
		s.logger.Panic("cannot decode message")
	}
	if envelope.ClientId == "" {
		// todo:
		// return bfttypes.RequestInfo{}, fmt.Errorf("empty client id")
		s.logger.Panic("VerifyRequest: empty client id")
	}
	if envelope.RequestId == "" {
		// todo:
		// return bfttypes.RequestInfo{}, fmt.Errorf("empty request id")
		s.logger.Panic("VerifyRequest: empty request id")
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
	s.ledger.mu.Lock()
	defer s.ledger.mu.Unlock()
	// s.logger.Warnf("VerificationSequence: %d", s.ledger.Sequence)
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
		if r.ClientId == "" {
			// todo:
			// return bfttypes.RequestInfo{}, fmt.Errorf("empty client id")
			s.logger.Panic("VerifyRequest: empty client id")
		}
		if r.RequestId == "" {
			// todo:
			// return bfttypes.RequestInfo{}, fmt.Errorf("empty request id")
			s.logger.Panic("VerifyRequest: empty request id")
		}
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
