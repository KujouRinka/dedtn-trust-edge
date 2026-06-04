from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class FrameRequest(_message.Message):
    __slots__ = ("image_data", "camera_id")
    IMAGE_DATA_FIELD_NUMBER: _ClassVar[int]
    CAMERA_ID_FIELD_NUMBER: _ClassVar[int]
    image_data: bytes
    camera_id: int
    def __init__(self, image_data: _Optional[bytes] = ..., camera_id: _Optional[int] = ...) -> None: ...

class DetectReply(_message.Message):
    __slots__ = ("boxes",)
    BOXES_FIELD_NUMBER: _ClassVar[int]
    boxes: _containers.RepeatedCompositeFieldContainer[Box]
    def __init__(self, boxes: _Optional[_Iterable[_Union[Box, _Mapping]]] = ...) -> None: ...

class Box(_message.Message):
    __slots__ = ("class_name", "class_id", "confidence", "x1", "y1", "x2", "y2")
    CLASS_NAME_FIELD_NUMBER: _ClassVar[int]
    CLASS_ID_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    X1_FIELD_NUMBER: _ClassVar[int]
    Y1_FIELD_NUMBER: _ClassVar[int]
    X2_FIELD_NUMBER: _ClassVar[int]
    Y2_FIELD_NUMBER: _ClassVar[int]
    class_name: str
    class_id: int
    confidence: float
    x1: float
    y1: float
    x2: float
    y2: float
    def __init__(self, class_name: _Optional[str] = ..., class_id: _Optional[int] = ..., confidence: _Optional[float] = ..., x1: _Optional[float] = ..., y1: _Optional[float] = ..., x2: _Optional[float] = ..., y2: _Optional[float] = ...) -> None: ...
