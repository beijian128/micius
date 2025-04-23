package netframe

import (
	"github/beijian128/micius/frame/util"
	"google.golang.org/protobuf/reflect/protoregistry"
	"testing"
)

func TestStringHash4(t *testing.T) {

	file := "netframe/message.proto"
	fd, err := protoregistry.GlobalFiles.FindFileByPath(file)
	if err != nil {
		return
	}

	mds := fd.Messages()
	for i := mds.Len() - 1; i >= 0; i-- {
		x := mds.Get(i)
		fullName := x.FullName()
		_, err := protoregistry.GlobalTypes.FindMessageByName(fullName)
		if err != nil {
			return
		}
		msgID := util.StringHash(string(fullName))
		t.Log(fullName, msgID)
	}
}
