package client

import (
	"log/slog"
	"net"
	"snake-game/internal/application/utils"
	snakespb "snake-game/internal/infrastructure/network/proto"
	"time"
)

// HandlePacket — центральный обработчик входящих сообщений протокола.
// Сюда Transport отдаёт уже распарсенный GameMessage + адрес отправителя.
func (c *Client) HandlePacket(msg *snakespb.GameMessage, addr *net.UDPAddr) {
	// Вне игры принимаем только Announcement (чтобы видеть список игр).
	// Остальные типы игнорируем, чтобы “вышедший master” не продолжал отвечать.
	if !c.isInGame() {
		switch msg.Type.(type) {
		case *snakespb.GameMessage_Announcement:
			c.handleAnnouncement(msg.GetAnnouncement(), addr)
			if c.view != nil {
				c.view.RefreshGamesList()
				c.view.RefreshAvailableGames()
			}
		case *snakespb.GameMessage_Discover:
			// вне игры не отвечаем анонсом
		default:
			// не отвечаем ack/ping/etc, чтобы "вышедший master" не жил
		}
		return
	}

	// Мы в игре: обновляем мониторинг “кто жив” ===
	// Любое сообщение от игрока означает, что он “жив”.
	senderID := msg.GetSenderId()
	if senderID != 0 {
		c.notePlayerHeard(senderID)
	}
	// Если мы MASTER — запоминаем адрес отправителя (нужно для рассылки state)
	if c.node.SelfRole == snakespb.NodeRole_MASTER {
		if sid := msg.GetSenderId(); sid != 0 && addr != nil {
			c.storePlayerAddress(sid, addr)
		}
	}
	slog.Info("RECV", "from", addr, "seq", msg.GetMsgSeq(), "type", utils.MsgTypeName(msg))
	//Роутинг по типу сообщения ===
	switch m := msg.Type.(type) {
	case *snakespb.GameMessage_Announcement:
		c.handleAnnouncement(m.Announcement, addr)
		if c.view != nil {
			c.view.RefreshGamesList()
			c.view.RefreshAvailableGames()
		}

	case *snakespb.GameMessage_Discover:
		// Discover обслуживает только MASTER: ответить анонсом в unicast
		if c.node.SelfRole != snakespb.NodeRole_MASTER {
			return
		}
		msg := c.buildAnnouncementMessage()
		if msg == nil {
			return
		}
		_ = c.clientTransport.SendTo(msg, addr)

	case *snakespb.GameMessage_Join:
		// Join обслуживает только MASTER: пытаемся добавить игрока
		playerID, err := c.HandleJoin(m.Join, addr)
		if err != nil {
			ack := c.buildAckMessage(msg.GetMsgSeq(), 0)
			if err2 := c.clientTransport.SendTo(ack, addr); err2 == nil {
				// id игрока ещё нет – некого отмечать
			}

			errMsg := c.buildErrorMessage(msg.GetMsgSeq(), err.Error())
			if errMsg != nil {
				_ = c.clientTransport.SendTo(errMsg, addr)
			}
			return
		}
		// Успешный Join: отправляем Ack, в receiver_id кладём присвоенный playerID
		ack := c.buildAckMessage(msg.GetMsgSeq(), playerID)
		if err := c.clientTransport.SendTo(ack, addr); err == nil {
			c.markSent(playerID)
		}

	case *snakespb.GameMessage_Ack:
		// Ack закрывает pending (надёжная доставка).
		// Если Ack на Join — тут мы узнаём master и свой id.
		if c.node.SelfID == 0 && msg.GetReceiverId() != 0 {
			c.node.MasterID = msg.GetSenderId()
			c.node.MasterAddr = utils.CopyUDPAddr(addr)
		}
		c.handleAck(msg)

	case *snakespb.GameMessage_Error:
		ack := c.buildAckMessage(msg.GetMsgSeq(), c.node.SelfID)
		if err := c.clientTransport.SendTo(ack, addr); err == nil {
			c.markSent(msg.GetSenderId())
		}
		c.handleError(msg, m, addr)

	case *snakespb.GameMessage_State:
		ack := c.buildAckMessage(msg.GetMsgSeq(), c.node.SelfID)
		_ = c.clientTransport.SendTo(ack, addr)
		c.markSent(msg.GetSenderId())

		c.handleState(msg, m.State, addr)

	case *snakespb.GameMessage_Steer:
		ack := c.buildAckMessage(msg.GetMsgSeq(), msg.GetSenderId())
		if err := c.clientTransport.SendTo(ack, addr); err == nil {
			c.markSent(msg.GetSenderId())
		}
		c.handleSteer(m.Steer, msg)

	case *snakespb.GameMessage_Ping:
		ack := c.buildAckMessage(msg.GetMsgSeq(), c.node.SelfID)
		if err := c.clientTransport.SendTo(ack, addr); err == nil {
			c.markSent(msg.GetSenderId())
		}
		c.notePlayerHeard(msg.GetSenderId())
	case *snakespb.GameMessage_RoleChange:
		// RoleChange подтверждаем Ack и применяем смену ролей (master/deputy/viewer)
		ack := c.buildAckMessage(msg.GetMsgSeq(), c.node.SelfID)
		_ = c.clientTransport.SendTo(ack, addr)
		c.markSent(msg.GetSenderId())

		c.handleRoleChange(msg, m.RoleChange, addr)
	default:
		slog.Info("unknown msg type:", m)
	}
}

// notePlayerHeard — обновляем lastHeard (используется мониторингом для timeout)
func (c *Client) notePlayerHeard(id int32) {
	if id == 0 {
		return
	}
	c.playerAddrsMu.Lock()
	defer c.playerAddrsMu.Unlock()

	if c.lastHeard == nil {
		c.lastHeard = make(map[int32]time.Time)
	}
	c.lastHeard[id] = time.Now()
}
