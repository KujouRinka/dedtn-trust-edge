package smart_bft

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	smartbft "github.com/hyperledger-labs/SmartBFT/pkg/consensus"
	bfttypes "github.com/hyperledger-labs/SmartBFT/pkg/types"
	"github.com/hyperledger-labs/SmartBFT/smartbftprotos"
	"github.com/kujourinka/dedtn-trust-edge/consensus"
	"github.com/kujourinka/dedtn-trust-edge/logger"
	"github.com/kujourinka/dedtn-trust-edge/semantic"
	"google.golang.org/grpc"
)

type Node struct {
	peers  []*peer
	server *RpcServer

	consensus *smartbft.Consensus

	verifier interface{}
	signer   interface{}

	clock       *time.Ticker
	secondClock *time.Ticker
}

func NewSmartPBFServer(config *Config) (consensus.Node, error) {
	server, err := NewRpcServer(config.ListenAddr, config.ListenPort)
	if err != nil {
		return nil, err
	}

	peers := make([]*peer, 0, len(config.Peers))
	for _, p := range config.Peers {
		np, err := newPeer(p.Id, p.Addr, p.Port)
		if err != nil {
			return nil, err
		}
		peers = append(peers, np)
	}
	node := &Node{
		peers:       peers,
		server:      server,
		clock:       time.NewTicker(time.Second),
		secondClock: time.NewTicker(time.Second),
	}

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
		// WAL:         writeAheadLog,
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
		if err := grpcServer.Serve(listener); err != nil {
			logger.Logger.Warnf("grpc server closed: %v", err)
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
	return s.SubmitRequest(msg.ToBytes())
}

func (s *Node) Stop() error {
	return nil
}

// Synchronizer interface

func (s *Node) Sync() bfttypes.SyncResponse {
	panic("unimplemented")
}

// RequestInspector interface

func (s *Node) RequestID(req []byte) bfttypes.RequestInfo {
	// Should return client id and request id
	return bfttypes.RequestInfo{}
}

// Application interface

// Deliver delivers the given proposal and signatures.
// After the call returns we assume that this proposal is stored in persistent memory.
// It returns whether this proposal was a reconfiguration and the current config.
func (s *Node) Deliver(proposal bfttypes.Proposal, signature []bfttypes.Signature) bfttypes.Reconfig {
	// if is trustable message then do on-chain
	// if is fake message, we should call a smart contract to recalculate trust score
	// Should we do these in a goroutine?
	panic("unimplemented")
}

// Assembler interface

func (s *Node) AssembleProposal(metadata []byte, requests [][]byte) bfttypes.Proposal {
	// assemble proposal from other node to bfttype.Proposal
	// only primary be called (?)
	// get message from SendTransaction (?)
	panic("unimplemented")
}

// Comm interface
func (s *Node) SendConsensus(targetID uint64, m *smartbftprotos.Message) {
	// send SmartBFT protocol message, such as prepare, commit
	// send this message to other node
	// other nodes received this message should call consensus.HandleMesssage()
	panic("unimplemented")
}

func (s *Node) SendTransaction(targetID uint64, request []byte) {
	// forward client request to other node, usually primary node
	panic("unimplemented")
}

func (s *Node) Nodes() []uint64 {
	// return all node in consensus currently
	ids := make([]uint64, 0, len(s.peers))
	for _, peer := range s.peers {
		ids = append(ids, peer.Id())
	}
	ids = append(ids, s.consensus.Config.SelfID)
	return ids
}

// Signer
func (s *Node) Sign(msg []byte) []byte {
	panic("unimplemented")
}

func (s *Node) SignProposal(proposal bfttypes.Proposal, auxiliaryInput []byte) *bfttypes.Signature {
	panic("unimplemented")
}

// Verifier interface

func (s *Node) VerifyProposal(proposal bfttypes.Proposal) ([]bfttypes.RequestInfo, error) {
	panic("unimplemented")
}

// VerifyRequest verifies the given request and returns its info.
func (s *Node) VerifyRequest(val []byte) (bfttypes.RequestInfo, error) {

	panic("unimplemented")
}

// VerifyConsenterSig verifies the signature for the given proposal.
// It returns the auxiliary data in the signature.
func (s *Node) VerifyConsenterSig(signature bfttypes.Signature, prop bfttypes.Proposal) ([]byte, error) {
	panic("unimplemented")
}

// VerifySignature verifies the signature.
func (s *Node) VerifySignature(signature bfttypes.Signature) error {
	panic("unimplemented")
}

// VerificationSequence returns the current verification sequence.
func (s *Node) VerificationSequence() uint64 {
	// return 0
	panic("unimplemented")
}

// RequestsFromProposal returns from the given proposal the included requests' info
func (s *Node) RequestsFromProposal(proposal bfttypes.Proposal) []bfttypes.RequestInfo {
	panic("unimplemented")
}

// AuxiliaryData extracts the auxiliary data from a signature's message
func (s *Node) AuxiliaryData([]byte) []byte {
	panic("unimplemented")
}

// MembershipNotifier interface

func (s *Node) MembershipChange() bool {
	return false
}

// Other
func (s *Node) HandleMessage(sender uint64, req []byte) {
	s.consensus.HandleRequest(sender, req)
}

func (s *Node) HandleRequest(sender uint64, req []byte) {
	s.consensus.HandleRequest(sender, req)
}

func (s *Node) SubmitRequest(req []byte) error {
	return s.consensus.SubmitRequest(req)
}
