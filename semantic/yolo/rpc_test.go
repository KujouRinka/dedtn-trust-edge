package yolo

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type mockServer struct {
	UnimplementedYoloServiceServer
	close chan struct{}
}

func newMockServer() *mockServer {
	return &mockServer{
		close: make(chan struct{}, 1),
	}
}

func (s *mockServer) CopyAndPaste(ctx context.Context, in *FrameRequest) (*FrameRequest, error) {
	s.close <- struct{}{}
	// time.Sleep(2 * time.Second)
	return &FrameRequest{
		ImageData: in.ImageData,
		CameraId:  in.CameraId,
	}, nil
}

// func (s *mockServer) DetectFrame(ctx context.Context, in *FrameRequest) (*DetectReply, error) {
// 	return nil, nil
// }

type mockClient struct {
	YoloServiceClient

	closer func() error
}

func newMockClient(host string, port uint16) (*mockClient, error) {
	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:%d", host, port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create client: %v", err)
	}
	client := NewYoloServiceClient(conn)

	return &mockClient{
		YoloServiceClient: client,
		closer:            conn.Close,
	}, nil
}

func (c *mockClient) Close() error {
	return c.closer()
}

func TestRpcSend(t *testing.T) {
	var wg sync.WaitGroup

	wg.Add(1)
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 23333))
	if err != nil {
		t.Fatal("cannot create listener:", err)
	}

	grpcServer := grpc.NewServer()
	s := newMockServer()
	RegisterYoloServiceServer(grpcServer, s)
	go func() {
		go func() {
			wg.Done()
			if err := grpcServer.Serve(listener); err != nil {
				t.Fatal("cannot start grpc server:", err)
			}
		}()
		<-s.close
		grpcServer.GracefulStop()
	}()

	wg.Wait()

	data, err := os.ReadFile("./assets/hikari.jpeg")
	if err != nil {
		t.Fatal("cannot open file:", err)
	}

	client, err := newMockClient("localhost", 23333)
	if err != nil {
		t.Fatal("cannot create client:", err)
	}
	defer client.Close()

	resp, err := client.CopyAndPaste(context.Background(), &FrameRequest{ImageData: data, CameraId: 114514})
	if err != nil {
		t.Fatal("rpc failed:", err)
	}

	if bytes.Compare(resp.ImageData, data) != 0 {
		t.Error("grpc data not consistence: ImageData")
	}
	if resp.CameraId != 114514 {
		t.Error("grpc data not consistence: CameraId")
	}
}

func TestRpcSendToPython(t *testing.T) {
	data, err := os.ReadFile("./assets/bus.jpg")
	if err != nil {
		t.Fatal("cannot open file:", err)
	}

	client, err := newMockClient("localhost", 23334)
	if err != nil {
		t.Fatal("cannot create client, have you launched python server?:", err)
	}

	defer client.Close()

	resp, err := client.CopyAndPaste(context.Background(), &FrameRequest{ImageData: data, CameraId: 114514})
	if err != nil {
		t.Fatal("rpc failed:", err)
	}

	if bytes.Compare(resp.ImageData, data) != 0 {
		t.Error("grpc data not consistence: ImageData")
	}
	if resp.CameraId != 114514 {
		t.Error("grpc data not consistence: CameraId")
	}

	predict, err := client.DetectFrame(context.Background(), &FrameRequest{ImageData: data, CameraId: 233})
	if err != nil {
		t.Fatal("rpc failed:", err)
	}

	for _, box := range predict.Boxes {
		if box.ClassId != 0 {
			// t.Error("box id not consistence")
		}
		t.Logf("%v", box)
	}
}
