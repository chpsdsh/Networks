package client

import (
	"log/slog"
	"net"
	"snake-game/internal/application/utils"
	snakespb "snake-game/internal/infrastructure/network/proto"
)

func (c *Client) handleRoleChange(
	msg *snakespb.GameMessage,
	rc *snakespb.GameMessage_RoleChangeMsg,
	addr *net.UDPAddr,
) {
	if rc == nil || msg == nil {
		return
	}

	oldSelfRole := c.node.SelfRole

	senderID := msg.GetSenderId()
	receiverID := msg.GetReceiverId()
	senderRole := rc.GetSenderRole()
	receiverRole := rc.GetReceiverRole()

	// master может обновлять адрес по любому unicast
	if c.node.SelfRole == snakespb.NodeRole_MASTER && senderID != 0 && addr != nil {
		c.storePlayerAddress(senderID, addr)
	}

	// Новый мастер объявился
	if senderRole == snakespb.NodeRole_MASTER && senderID != 0 && addr != nil {
		if c.node.MasterID != senderID {
			oldMaster := c.node.MasterID

			slog.Info("ROLECHANGE: new master detected", "id", senderID, "addr", addr)

			c.node.MasterID = senderID
			c.node.MasterAddr = utils.CopyUDPAddr(addr)

			c.node.DeputyID = 0
			c.node.DeputyAddr = nil

			if oldMaster != 0 {
				c.reroutePending(oldMaster, c.node.MasterID, c.node.MasterAddr)
			}

			// если новый master НЕ я
			if senderID != c.node.SelfID {
				c.stopAnnouncement()
			}
		}
	}

	//  Master  назначил наc deputy
	if receiverID == c.node.SelfID && receiverRole == snakespb.NodeRole_DEPUTY {
		if c.node.SelfRole == snakespb.NodeRole_VIEWER {
			slog.Info("ROLECHANGE: ignore deputy assignment because I am VIEWER")
			return
		}

		slog.Info("ROLECHANGE: I am DEPUTY now ", "master", senderID)
		c.node.SelfRole = snakespb.NodeRole_DEPUTY
		c.node.MasterID = senderID
		c.node.MasterAddr = utils.CopyUDPAddr(addr)

		// deputy не шлёт announcement
		c.stopAnnouncement()
		return
	}

	//Кик/выход в viewer
	if receiverID == c.node.SelfID && receiverRole == snakespb.NodeRole_VIEWER {
		slog.Info("ROLECHANGE: I am VIEWER now")
		c.node.SelfRole = snakespb.NodeRole_VIEWER

		//viewer тоже не шлёт announcement
		c.stopAnnouncement()
		return
	}

	//если раньше я был MASTER, а теперь нет — гасим announcement
	if oldSelfRole == snakespb.NodeRole_MASTER && c.node.SelfRole != snakespb.NodeRole_MASTER {
		c.stopAnnouncement()
	}
}
