#!/bin/zsh

# create python venv
python3 -m venv .venv
source .venv/bin/activate
python3 -m pip install grpcio
python3 -m pip install grpcio-tools
python3 -m pip install ultralytics
python3 -m pip install opencv-python ultralytics numpy

# compile proto for rpc server
python3 -m grpc_tools.protoc \
  -I./semantic/yolo \
  --python_out=./semantic/yolo \
  --pyi_out=./semantic/yolo \
  --grpc_python_out=./semantic/yolo \
  ./semantic/yolo/rpc.proto

# install gstreamer
case "$(uname)" in
  Darwin)
    brew install gstreamer
    brew install gst-plugins-base
    brew install gst-plugins-good
    brew install gst-plugins-bad
    brew install gst-libav

    brew install protobuf
    ;;
  Linux)
    echo "Linux"
    ;;
  *)
    echo "Other"
    ;;
esac

go generate proto.go

protoc -I=./semantic/yolo \
  --go_out=./semantic/yolo \
  --go_opt=paths=source_relative \
  --go-grpc_out=./semantic/yolo \
  --go-grpc_opt=paths=source_relative ./semantic/yolo/rpc.proto \
  --plugin=protoc-gen-go=$(go env GOPATH)/bin/protoc-gen-go \
  --plugin=protoc-gen-go-grpc=$(go env GOPATH)/bin/protoc-gen-go-grpc
