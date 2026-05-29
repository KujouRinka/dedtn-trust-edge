#!/bin/zsh

source .venv/bin/activate
python3 ./semantic/yolo/server.py --host localhost --port 23334
