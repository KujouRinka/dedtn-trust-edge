#!/bin/zsh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_DIR=${SCRIPT_DIR}/..

source ${PROJ_DIR}/.venv/bin/activate
python3 ${PROJ_DIR}/semantic/yolo/server.py --host localhost --port 23334
