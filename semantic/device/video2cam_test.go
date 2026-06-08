package device

import (
	"testing"
)

func TestVideo2Cam(t *testing.T) {
	virtualCam, err := NewVideo2Cam("./assets/video.mp4")
	if err != nil {
		t.Fatal("cannot create video2cam:", err)
	}
	defer virtualCam.Close()

	for i := 0; i < 50; i++ {
		frameData, err := virtualCam.CurrentFrame()
		if err != nil {
			t.Fatal("cannot read frame from camera:", err)
		}

		if frameData == nil {
			t.Fatal("cannot read frame from file")
		}
		// time.Sleep(5 * time.Second)
	}
}
