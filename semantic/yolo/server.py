import argparse
import sys
import logging

import cv2
import grpc
from concurrent.futures.thread import ThreadPoolExecutor
import numpy as np
from ultralytics import YOLO

import rpc_pb2
import rpc_pb2_grpc


logging.basicConfig(
  level=logging.INFO,
  format="%(asctime)s [%(levelname)s] %(message)s"
)
logger = logging.getLogger(__name__)

class YoloPredict(object):
  def __init__(self, model_name="yolo26n.pt", save=False):
    self._model = YOLO(model_name)
    self._save = save

  def predict(self, data_byte):
    nparr = np.frombuffer(data_byte, np.uint8)
    img = cv2.imdecode(nparr, cv2.IMREAD_COLOR)
    results = self._model.predict(img, save=self._save, name="detect_result")
    return results


class GrpcServicer(rpc_pb2_grpc.YoloServiceServicer):
  def __init__(self, save):
    self._predictor = YoloPredict("yolo26n.pt", save)

  def CopyAndPaste(self, request, context):
    logger.info("CopyAndPaste test")
    return rpc_pb2.FrameRequest(
      image_data=request.image_data,
      camera_id=request.camera_id,
    )

  def DetectFrame(self, request, context):
    logger.info("DetectFrame: from camera: {}".format(request.camera_id))
    result = self._predictor.predict(request.image_data)[0]
    ret = rpc_pb2.DetectReply()
    names = [result.names[cls.item()] for cls in result.boxes.cls.int()]  # class name of each box
    for i in range(len(result.boxes.cls)):
      b = rpc_pb2.Box(
        class_name=names[i],
        confidence=result.boxes.conf[i],
        class_id=result.boxes.cls.int()[i],
        x1=result.boxes.xywh[i][0],
        y1=result.boxes.xywh[i][1],
        x2=result.boxes.xywh[i][2],
        y2=result.boxes.xywh[i][3],
      )
      ret.boxes.append(b)
    return ret


class GrpcServer(object):
  def __init__(self, host, port, is_save):
    self._host = host
    self._port = port
    self._save = is_save

  def serve(self):
    server = grpc.server(ThreadPoolExecutor(max_workers=10))
    rpc_pb2_grpc.add_YoloServiceServicer_to_server(GrpcServicer(self._save), server)
    server.add_insecure_port(self._host + ":" + str(self._port))
    server.start()
    logger.info("YOLO RPC server up")
    server.wait_for_termination()


def main():
  parser = argparse.ArgumentParser()
  parser.add_argument("--host", type=str, default="localhost")
  parser.add_argument("--port", type=int, default=23334)
  parser.add_argument("--save", action="store_true")
  args = parser.parse_args()

  host, port = args.host, args.port
  serv = GrpcServer(host, int(port), args.save)
  serv.serve()


if __name__ == "__main__":
  main()
