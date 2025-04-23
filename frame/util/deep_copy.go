package util

import (
	"bytes"
	"encoding/gob"

	"github.com/golang/protobuf/proto"
)

// DeepCopy 深拷贝（未导出字段不拷贝）
func DeepCopy(dst, src any) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}

// DeepCopyUseProtobuf ...
func DeepCopyUseProtobuf(dst proto.Message, src proto.Message) error {
	data, err := proto.Marshal(src)
	if err != nil {
		return err
	}
	return proto.Unmarshal(data, dst)
}
