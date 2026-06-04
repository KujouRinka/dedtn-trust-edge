package smart_bft

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockData struct {
	Foo int `json:"foo"`
	Bar int `json:"bar"`
}

func (m *mockData) Digest() []byte {
	return nil
}

func (m *mockData) ToBytes() []byte {
	b, err := json.Marshal(m)
	if err != nil {
		panic("encode json error")
	}
	return b
}

func TestConsensus(t *testing.T) {
	nodeCount := 10

	peersForEach := make([][]*PeerCfg, nodeCount)
	for i := 0; i < nodeCount; i++ {
		peersForEach[i] = make([]*PeerCfg, 0, nodeCount-1)
		for j := 0; j < nodeCount; j++ {
			if j == i {
				continue
			}
			peer := &PeerCfg{
				Id:   uint64(j + 1),
				Addr: "localhost",
				Port: uint16(23333 + i),
			}
			peersForEach[i] = append(peersForEach[i], peer)
		}
	}

	testDir, err := os.MkdirTemp("", "naive_chain")
	assert.NoErrorf(t, err, "generate temporary test dir")
	defer os.RemoveAll(testDir)

	nodes := make([]*Node, 0, nodeCount)
	for i := 0; i < nodeCount; i++ {
		cfg := &Config{
			Id:         uint64(i + 1),
			NodeDir:    testDir,
			ListenAddr: "localhost",
			ListenPort: uint16(23333 + i),
			Peers:      peersForEach[i],
		}
		node, err := NewSmartPBFServer(cfg)
		if err != nil {
			t.Fatal("cannot create node:", err)
		}
		n, ok := node.(*Node)
		if !ok {
			t.Fatal("cannot convert")
		}
		nodes = append(nodes, n)
	}

	for _, node := range nodes {
		go func(node *Node) {
			defer node.Stop()
			if err := node.Run(); err != nil {
				t.Error("start node failed:", err)
			}
		}(node)
	}

	for i := 0; i < nodeCount*100; i++ {
		if err := nodes[i%nodeCount].SubmitSemantic(&mockData{i + 42, i + 2333}); err != nil {
			t.Fatal("submit message error:", err)
		}
	}

	for _, node := range nodes {
		node.Stop()
	}
}
