package semantic

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/kujourinka/dedtn-trust-edge/semantic/device"
	"github.com/kujourinka/dedtn-trust-edge/semantic/yolo"
)

type ImgSemanticGen struct {
	msg chan SemanticClaim
	ctx context.Context

	yoloServer    *yolo.RpcClient
	camera        device.Camera
	watchInterval time.Duration

	io.Closer
}

func NewImgSemanticGen(cfg *ImageSemanticConfig, ctx context.Context) (*ImgSemanticGen, error) {
	yoloServer, err := yolo.NewRpcClient(cfg.RpcHost, cfg.RpcPort)
	if err != nil {
		return nil, err
	}
	camera, err := device.NewVideo2Cam(cfg.Source)

	return &ImgSemanticGen{
		ctx:           ctx,
		msg:           make(chan SemanticClaim),
		yoloServer:    yoloServer,
		camera:        camera,
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

			resp, err := i.yoloServer.DetectFrame(context.Background(), &yolo.FrameRequest{
				ImageData: frame,
				CameraId:  0,
			})
			if err != nil {
				return err
			}

			i.msg <- SemanticClaim{Payload: resp.Boxes}
			time.Sleep(i.watchInterval)
		}
	}
}

func (i *ImgSemanticGen) ReadChan() <-chan SemanticClaim {
	return i.msg
}

func (i *ImgSemanticGen) Close() error {
	var errs []error
	if err := i.camera.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := i.yoloServer.Close(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
