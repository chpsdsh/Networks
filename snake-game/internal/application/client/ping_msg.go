package client

import (
	snakespb "snake-game/internal/infrastructure/network/proto"

	"google.golang.org/protobuf/proto"
)

func (c *Client) sendPingToPlayer(id int32) {
	c.playerAddrsMu.RLock()
	addr := c.playersAddr[id]
	c.playerAddrsMu.RUnlock()

	if addr == nil {
		return
	}

	msg := &snakespb.GameMessage{
		SenderId:   proto.Int32(c.node.SelfID),
		ReceiverId: proto.Int32(id),
		Type: &snakespb.GameMessage_Ping{
			Ping: &snakespb.GameMessage_PingMsg{},
		},
	}

	// Пинг можно не очень надёжно, 1 попытка достаточно
	if err := c.clientTransport.SendTo(msg, addr); err == nil {
		c.markSent(id)
	}
}
