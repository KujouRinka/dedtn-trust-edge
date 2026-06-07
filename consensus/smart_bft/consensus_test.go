package smart_bft

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/kujourinka/dedtn-trust-edge/logger"
	semantic2 "github.com/kujourinka/dedtn-trust-edge/semantic"
	"github.com/kujourinka/dedtn-trust-edge/semantic/yolo"
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

func (m *mockData) Data() interface{} {
	return nil
}

func TestConsensus(t *testing.T) {
	logger.LogLevel = "error"
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
				Port: uint16(23333 + j),
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

	var wg sync.WaitGroup
	errChan := make(chan error, nodeCount)
	for _, node := range nodes {
		wg.Add(1)
		go func(node *Node) {
			defer node.Stop()
			// todo: fix this
			wg.Done()
			if err := node.Run(); err != nil {
				errChan <- fmt.Errorf("start %s failed: %v", node.idName, err)
			}
		}(node)
	}

	// todo: fix this
	wg.Wait()

	select {
	case <-time.Tick(1 * time.Second):
	case err := <-errChan:
		t.Fatalf("start node failed: %v", err)
	}

	// generate mock data
	var semantic semantic2.SemanticClaim
	var detectReply yolo.DetectReply
	detectReply.Boxes = make([]*yolo.Box, 4)
	for i := 0; i < 4; i++ {
		detectReply.Boxes[i] = new(yolo.Box)
	}

	var deliverWg sync.WaitGroup
	deliverWg.Add(nodeCount)
	deliverCnt := nodeCount * 100
	for i := 0; i < deliverCnt; i++ {
		for j := 0; j < 4; j++ {
			detectReply.Boxes[j].ClassName = "class" + strconv.FormatInt(int64(i*j), 10)
			detectReply.Boxes[j].ClassId = int32(i * j)
			detectReply.Boxes[j].Confidence = (float32)(i*j%100) / 100
		}

		semantic.Payload = &detectReply
		if err := nodes[i%nodeCount].SubmitSemantic(&semantic); err != nil {
			t.Fatal(fmt.Sprintf("Node%d: submit message error:", nodes[i%nodeCount].consensus.Config.SelfID), err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	for i := 0; i < nodeCount; i++ {
		go func(ctx context.Context) {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			defer deliverWg.Done()

			for {
				if nodes[i].deliverCount.Load() == uint64(deliverCnt) {
					return
				}
				select {
				case <-ctx.Done():
					t.Errorf("node%d deliver timeout", ctx.Err())
					return
				case <-ticker.C:
				}
			}
		}(ctx)
	}

	deliverWg.Wait()

	for _, node := range nodes {
		if err := node.Stop(); err != nil {
			t.Error("stop node failed:", err)
		}
	}
	for i := 0; i < nodeCount; i++ {
		t.Logf("%s deliver count: %d", nodes[i].idName, nodes[i].deliverCount.Load())
	}
}
