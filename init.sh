#!/bin/zsh

go get github.com/ethereum/go-ethereum@1.13.15

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

./scripts/compile_protos.sh
