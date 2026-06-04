#!/bin/sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_DIR=${SCRIPT_DIR}/..

# create python venv
python3 -m venv ${PROJ_DIR}/.venv
source ${PROJ_DIR}/.venv/bin/activate
python3 -m pip install grpcio
python3 -m pip install grpcio-tools
python3 -m pip install ultralytics
python3 -m pip install opencv-python ultralytics numpy

# compile proto for rpc server
python3 -m grpc_tools.protoc \
  -I${PROJ_DIR}/semantic/yolo \
  --python_out=${PROJ_DIR}/semantic/yolo \
  --pyi_out=${PROJ_DIR}/semantic/yolo \
  --grpc_python_out=${PROJ_DIR}/semantic/yolo \
  ${PROJ_DIR}/semantic/yolo/yolo_rpc.proto

go generate ${SCRIPT_DIR}/proto.go

protoc -I=${PROJ_DIR}/semantic/yolo \
  --go_out=${PROJ_DIR}/semantic/yolo \
  --go_opt=paths=source_relative \
  --go-grpc_out=${PROJ_DIR}/semantic/yolo \
  --go-grpc_opt=paths=source_relative ${PROJ_DIR}/semantic/yolo/yolo_rpc.proto \
  --plugin=protoc-gen-go=$(go env GOPATH)/bin/protoc-gen-go \
  --plugin=protoc-gen-go-grpc=$(go env GOPATH)/bin/protoc-gen-go-grpc

protoc -I=${PROJ_DIR}/consensus/smart_bft \
  -I "$(go env GOMODCACHE)/github.com/hyperledger-labs/!smart!b!f!t@v1.0.1/smartbftprotos" \
  --go_out=${PROJ_DIR}/consensus/smart_bft \
  --go_opt=paths=source_relative \
  --go-grpc_out=${PROJ_DIR}/consensus/smart_bft \
  --go-grpc_opt=paths=source_relative ${PROJ_DIR}/consensus/smart_bft/smartbft_rpc.proto \
  --plugin=protoc-gen-go=$(go env GOPATH)/bin/protoc-gen-go \
  --plugin=protoc-gen-go-grpc=$(go env GOPATH)/bin/protoc-gen-go-grpc

