package netframe

import (
	"github/beijian128/micius/frame/util"
	"google.golang.org/protobuf/reflect/protoregistry"
	"testing"
)

func Test_initMessage(t *testing.T) {

	targetMsgID := 662840471
	for _, file := range []string{
		"netframe/testmsg.proto",
		"netframe/message.proto",
	} {
		fd, err := protoregistry.GlobalFiles.FindFileByPath(file)
		if err != nil {
			continue
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
			if msgID == uint32(targetMsgID) {
				t.Log(fullName, msgID)
				return
			}
		}
	}

}
