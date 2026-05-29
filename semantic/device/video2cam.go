package device

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/go-gst/go-gst/gst"
	"github.com/go-gst/go-gst/gst/app"
)

type video2cam struct {
	filename   string
	createTime time.Time

	pipeline      *gst.Pipeline
	sink          *app.Sink
	videoDuration time.Duration

	mu sync.Mutex
}

func NewVideo2Cam(filename string) (Camera, error) {
	gst.Init(nil)

	uri, err := filePathToURI(filename)
	if err != nil {
		return nil, err
	}

	width := 800
	height := 600
	pipelineDesc := fmt.Sprintf(
		"uridecodebin uri=%s ! "+
			"videoconvert ! "+
			"videoscale ! "+
			"video/x-raw,width=%d,height=%d ! "+
			"jpegenc quality=90 ! "+
			"appsink name=sink sync=false max-buffers=1 drop=true",
		uri,
		width,
		height,
	)

	pipeline, err := gst.NewPipelineFromString(pipelineDesc)
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}

	sinkElem, err := pipeline.GetElementByName("sink")
	if err != nil {
		_ = pipeline.SetState(gst.StateNull)
		return nil, fmt.Errorf("failed to get appsink: %w", err)
	}

	sink := app.SinkFromElement(sinkElem)
	if sink == nil {
		_ = pipeline.SetState(gst.StateNull)
		return nil, fmt.Errorf("failed to cast element to appsink")
	}
	sink.SetMaxBuffers(1)
	sink.SetDrop(true)
	sink.SetWaitOnEOS(false)

	if err := pipeline.BlockSetState(gst.StatePaused); err != nil {
		_ = pipeline.SetState(gst.StateNull)
		return nil, fmt.Errorf("failed to set pipeline paused: %w", err)
	}

	ok, durationNs := pipeline.QueryDuration(gst.FormatTime)
	if !ok || durationNs <= 0 {
		_ = pipeline.SetState(gst.StateNull)
		return nil, fmt.Errorf("failed to query video duration")
	}

	return &video2cam{
		filename:      filename,
		createTime:    time.Now(),
		pipeline:      pipeline,
		sink:          sink,
		videoDuration: time.Duration(durationNs),
	}, nil
}

func (c *video2cam) CurrentFrame() ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elapsed := time.Since(c.createTime)
	position := elapsed % c.videoDuration
	flag := gst.SeekFlagFlush | gst.SeekFlagAccurate

	ok := c.pipeline.SeekTime(position, flag)
	if !ok {
		return nil, fmt.Errorf("failed to seek to %v", position)
	}

	sample := c.sink.TryPullPreroll(gst.ClockTime(3 * time.Second))
	if sample == nil {
		return nil, fmt.Errorf("failed to pull jpeg frame at %v", position)
	}

	buffer := sample.GetBuffer()
	if buffer == nil {
		return nil, fmt.Errorf("sample has no buffer")
	}

	size, _, _ := buffer.GetSizes()
	imgBytes := buffer.Extract(0, size)

	runtime.KeepAlive(buffer)
	runtime.KeepAlive(sample)

	return imgBytes, nil
}

func (c *video2cam) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.pipeline != nil {
		if err := c.pipeline.SetState(gst.StateNull); err != nil {
			return err
		}
		c.pipeline = nil
		c.sink = nil
	}
	return nil
}

func filePathToURI(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("video file not found: %s, err: %w", abs, err)
	}

	u := url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(abs),
	}

	return u.String(), nil
}
