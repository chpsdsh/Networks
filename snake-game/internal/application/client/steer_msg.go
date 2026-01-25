package client

import (
	"snake-game/internal/application/utils"
	"snake-game/internal/domain"
	snakespb "snake-game/internal/infrastructure/network/proto"

	"google.golang.org/protobuf/proto"
)

func (c *Client) handleSteer(steer *snakespb.GameMessage_SteerMsg, msg *snakespb.GameMessage) {
	if c.node.SelfRole != snakespb.NodeRole_MASTER {
		return
	}
	if c.game == nil || steer == nil {
		return
	}

	playerID := msg.GetSenderId()
	if playerID == 0 {
		return
	}

	dir, ok := utils.FromProtoDirection(steer.GetDirection())
	if !ok {
		return
	}

	seq := msg.GetMsgSeq()

	c.steerMu.Lock()
	defer c.steerMu.Unlock()

	if c.lastSteerSeq == nil {
		c.lastSteerSeq = make(map[int32]int64)
	}
	if c.pendingSteer == nil {
		c.pendingSteer = make(map[int32]domain.Direction)
	}

	// В пределах тика принимаем только самый новый steer по msg_seq
	if prev := c.lastSteerSeq[playerID]; seq <= prev {
		return
	}

	c.lastSteerSeq[playerID] = seq
	c.pendingSteer[playerID] = dir
}

func (c *Client) buildSteerMessage(dir domain.Direction) *snakespb.GameMessage {
	var pdir snakespb.Direction
	switch dir {
	case domain.DirUp:
		pdir = snakespb.Direction_UP
	case domain.DirDown:
		pdir = snakespb.Direction_DOWN
	case domain.DirLeft:
		pdir = snakespb.Direction_LEFT
	case domain.DirRight:
		pdir = snakespb.Direction_RIGHT
	default:
		pdir = snakespb.Direction_UP
	}

	return &snakespb.GameMessage{
		SenderId: proto.Int32(c.node.SelfID),
		Type: &snakespb.GameMessage_Steer{
			Steer: &snakespb.GameMessage_SteerMsg{
				Direction: &pdir,
			},
		},
	}
}

func (c *Client) applyPendingSteer() {
	if c.node.SelfRole != snakespb.NodeRole_MASTER || c.game == nil {
		return
	}

	c.steerMu.Lock()
	defer c.steerMu.Unlock()

	for pid, dir := range c.pendingSteer {
		c.game.SetDirection(dir, pid)
	}

	// очищаем “тик-буфер”
	for k := range c.pendingSteer {
		delete(c.pendingSteer, k)
	}
	for k := range c.lastSteerSeq {
		delete(c.lastSteerSeq, k)
	}
}
