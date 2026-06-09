package semantic

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kujourinka/dedtn-trust-edge/semantic/types"
)

func TestImageGenerator(t *testing.T) {
	config := &types.ImageSemanticConfig{
		RpcHost:      "localhost",
		RpcPort:      30000,
		TimeInterval: 1,
		Source:       "./assets/video.mp4",
	}

	ctx, cancel := context.WithCancel(context.Background())
	gen, err := NewImgSemanticGen(config, ctx)
	if err != nil {
		t.Fatalf("cannot create generator: %v", err)
	}
	defer gen.Close()

	go func() {
		if err := gen.Run(); !errors.Is(err, context.Canceled) {
			t.Errorf("generator do not exit normally: %v", err)
		}
	}()

	go func() {
		time.Sleep(3 * time.Second)
		cancel()
	}()

	for semData := range gen.ReadChan() {
		t.Log(semData)
	}
}
