package device

import (
	"context"
	"testing"

	"github.com/kujourinka/dedtn-trust-edge/semantic/yolo"
)

func TestVideo2Cam(t *testing.T) {
	virtualCam, err := NewVideo2Cam("./assets/video.mp4")
	if err != nil {
		t.Fatal("cannot create video2cam:", err)
	}
	defer virtualCam.Close()

	yoloRpc, err := yolo.NewRpcClient("localhost", 23334)
	if err != nil {
		t.Fatal("cannot create yolo rpc client:", err)
	}
	defer yoloRpc.Close()

	for i := 0; i < 50; i++ {
		frameData, err := virtualCam.CurrentFrame()
		if err != nil {
			t.Fatal("cannot read frame from camera:", err)
		}

		resp, err := yoloRpc.DetectFrame(context.Background(), &yolo.FrameRequest{ImageData: frameData, CameraId: 42})
		if err != nil {
			t.Fatal("grpc failed:", err)
		}
		t.Logf("%v", resp)
		// time.Sleep(5 * time.Second)
	}
}
