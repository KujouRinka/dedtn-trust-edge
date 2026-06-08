package semantic

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/kujourinka/dedtn-trust-edge/semantic/device"
	"github.com/kujourinka/dedtn-trust-edge/semantic/types"
	"github.com/kujourinka/dedtn-trust-edge/semantic/yolo"
)

type ImgSemanticGen struct {
	msg chan types.Result
	ctx context.Context

	// todo: should be device
	camera        device.Camera
	semanticist   types.Semanticist
	watchInterval time.Duration

	io.Closer
}

func NewImgSemanticGen(cfg *types.ImageSemanticConfig, ctx context.Context) (*ImgSemanticGen, error) {
	// todo: create semanticist from config
	yoloServer, err := yolo.NewRpcClient(cfg.RpcHost, cfg.RpcPort)
	if err != nil {
		return nil, err
	}
	camera, err := device.NewVideo2Cam(cfg.Source)
	if err != nil {
		return nil, err
	}

	return &ImgSemanticGen{
		ctx:           ctx,
		msg:           make(chan types.Result),
		camera:        camera,
		semanticist:   yoloServer,
		watchInterval: cfg.TimeInterval,
	}, nil
}

func (i *ImgSemanticGen) Run() error {
	defer close(i.msg)

	for {
		select {
		case <-i.ctx.Done():
			return i.ctx.Err()
		default:
			frame, err := i.camera.CurrentFrame()
			if err != nil {
				log.Error("failed to get current frame:", err)
				continue
			}

			result, err := i.semanticist.Semanticize(context.Background(), frame)
			if err != nil {
				return err
			}

			i.msg <- result
			time.Sleep(i.watchInterval)
		}
	}
}

func (i *ImgSemanticGen) ReadChan() <-chan types.Result {
	return i.msg
}

func (i *ImgSemanticGen) Close() error {
	var errs []error
	if err := i.camera.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := i.semanticist.Close(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
