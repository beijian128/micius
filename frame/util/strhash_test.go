package util

import (
	_ "github/beijian128/micius/proto/cmsg"
	_ "github/beijian128/micius/proto/smsg"
	"google.golang.org/protobuf/reflect/protoregistry"
	"testing"
	"time"
)

func TestStringHash(t *testing.T) {

	file := "cmsg/client_msg.proto"
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
		msgID := StringHash(string(fullName))
		t.Log(fullName, msgID)
	}

}

func TestStringHashProtoName(t *testing.T) {

	file := "cmsg/client_msg.proto"
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
		msgID := StringHash(string(fullName))
		if msgID == 183035257 {
			t.Log(fullName, msgID)
		}

	}

}

func TestStringHashProtoName3(t *testing.T) {

	file := "smsg/gate_msg.proto"
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
		msgID := StringHash(string(fullName))
		if msgID == 3161802341 {
			t.Log(fullName, msgID)
		}

	}

}

func TestStringHash2(t *testing.T) {
	for i := 0; i < 10000; i++ {
		go func() {
			StringHash("cmsg.CReqsjihfid")
			a := 1
			b := 2
			if a+b == 4 {
				t.Log(a + b)
			}
		}()
	}
	time.Sleep(time.Second * 4)
}
