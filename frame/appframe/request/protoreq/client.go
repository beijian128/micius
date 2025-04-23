package protoreq

import (
	"errors"
	"github/beijian128/micius/frame/framework/netframe"
	"time"

	"github.com/golang/protobuf/proto"
	"github/beijian128/micius/frame/appframe/request"
)

// Client 用于发起消息请求, 接受响应
type Client struct {
	*request.Client
}

// NewClient 创建 Client
func NewClient(worker func(func())) *Client {
	return &Client{
		Client: request.NewClient(worker),
	}
}

// Req 发起请求, 异步回调, 请求消息必须有 Seqid int64 字段
func (c *Client) Req(sendMsg func(proto.Message, netframe.Server_Extend) error, msg proto.Message, extend netframe.Server_Extend, cbk func(resp proto.Message, err error), timeout time.Duration) (cancel func()) {
	return c.Client.Req(func(seqID int64) error {
		extend.SeqId = seqID
		return sendMsg(msg, extend)
	}, func(resp any, err error) {
		if err == nil {
			msg, ok := resp.(proto.Message)
			if ok {
				cbk(msg, err)
			} else {
				cbk(nil, errors.New("resp not a proto.Message"))
			}
		} else {
			cbk(nil, err)
		}
	}, timeout)
}

// Call 阻塞调用, 消息必须有 Seqid int64 字段
func (c *Client) Call(sendMsg func(proto.Message, netframe.Server_Extend) error, msg proto.Message, extend netframe.Server_Extend, timeout time.Duration) (proto.Message, error) {
	resp, err := c.Client.Call(func(seqID int64) error {
		extend.SeqId = seqID
		return sendMsg(msg, extend)
	}, timeout)
	if err != nil {
		return nil, err
	}
	msg, ok := resp.(proto.Message)
	if !ok {
		return nil, errors.New("resp not a proto.Message")
	}
	return msg, nil
}

// OnResp 响应消息, 消息必须有 Seqid int64 字段
func (c *Client) OnResp(msg proto.Message, seqID int64) {
	if err, ok := msg.(*ErrCode); ok {
		c.Client.OnErr(seqID, err)
	} else {
		c.Client.OnResp(seqID, msg)
	}
}
