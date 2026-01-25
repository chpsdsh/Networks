package client

import (
	"log/slog"
	snakespb "snake-game/internal/infrastructure/network/proto"

	"google.golang.org/protobuf/proto"
)

func (c *Client) buildAckMessage(
	origSeq int64,
	receiverID int32,
) *snakespb.GameMessage {

	return &snakespb.GameMessage{
		MsgSeq:     proto.Int64(origSeq),
		SenderId:   proto.Int32(c.node.SelfID),
		ReceiverId: proto.Int32(receiverID),
		Type: &snakespb.GameMessage_Ack{
			Ack: &snakespb.GameMessage_AckMsg{},
		},
	}
}

func (c *Client) handleAck(msg *snakespb.GameMessage) {
	ack := msg.GetAck()
	if ack == nil {
		return
	}

	// id, который мастер нам присвоил
	rid := msg.GetReceiverId()
	if rid != 0 && c.node.SelfID == 0 {
		c.node.SelfID = rid
		slog.Info("JOIN OK, my ", slog.Int("id ", int(rid)))
	}

	c.completePending(msg.GetMsgSeq(), nil)
}
