package client

import (
	"fmt"
	"snake-game/internal/application/utils"
	snakespb "snake-game/internal/infrastructure/network/proto"

	"net"
	"time"

	"google.golang.org/protobuf/proto"
)

// nextSeq — атомарно увеличивает и возвращает msg_seq
func (c *Client) nextSeq() int64 {
	c.seqMu.Lock()
	defer c.seqMu.Unlock()
	c.seq++
	return c.seq
}

// markSent — отмечаем, что мы что-то отправили игроку
func (c *Client) markSent(id int32) {
	if id == 0 {
		return
	}
	c.playerAddrsMu.Lock()
	defer c.playerAddrsMu.Unlock()

	if c.lastSent == nil {
		c.lastSent = make(map[int32]time.Time)
	}
	c.lastSent[id] = time.Now()
}

// protectedSend — асинхронная надёжная отправка:
//   - сразу возвращаемся
//   - ретраи делает reliability loop
func (c *Client) protectedSend(
	msg *snakespb.GameMessage,
	addr *net.UDPAddr,
	peerID int32,
	retry time.Duration,
	attempts int,
) (int64, <-chan error, error) {

	if addr == nil || msg == nil {
		return 0, nil, fmt.Errorf("nil addr/msg")
	}

	seq := msg.GetMsgSeq()
	if seq == 0 {
		seq = c.nextSeq()
		msg.MsgSeq = &seq
	}

	if retry <= 0 {
		retry = 100 * time.Millisecond
	}

	// pendingItem хранит всё для повторных отправок
	pi := &pendingItem{
		seq:          seq,
		msg:          proto.Clone(msg).(*snakespb.GameMessage), // копия!
		addr:         utils.CopyUDPAddr(addr),
		peerID:       peerID,
		retryEvery:   retry,
		nextSend:     time.Now(), // сразу отправить
		attemptsLeft: attempts,
		done:         make(chan error, 1),
	}

	c.pendingMu.Lock()
	if c.pending == nil {
		c.pending = make(map[int64]*pendingItem)
	}
	c.pending[seq] = pi
	c.pendingMu.Unlock()

	//первая отправка сразу
	c.enqueueNotProtected(pi.msg, pi.addr, pi.peerID)

	return seq, pi.done, nil
}

// protectedSendWait — удобный helper:
// асинхронная отправка + ожидание Ack с общим таймаутом
func (c *Client) protectedSendWait(
	msg *snakespb.GameMessage,
	addr *net.UDPAddr,
	peerID int32,
	retry time.Duration,
	attempts int,
	waitTotal time.Duration, // общий таймаут ожидания (например 2*stateDelay)
) error {
	_, done, err := c.protectedSend(msg, addr, peerID, retry, attempts)
	if err != nil {
		return err
	}

	if waitTotal <= 0 {
		waitTotal = 2 * time.Second
	}

	select {
	case e := <-done:
		return e
	case <-time.After(waitTotal):
		return fmt.Errorf("timeout waiting ack")
	}
}

// reroutePending — при смене мастера:
// все pending-сообщения старому мастеру
// перенаправляем новому мастеру
func (c *Client) reroutePending(oldPeerID, newPeerID int32, newAddr *net.UDPAddr) {
	if oldPeerID == 0 || newPeerID == 0 || newAddr == nil {
		return
	}

	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()

	for _, pi := range c.pending {
		if pi == nil {
			continue
		}
		if pi.peerID != oldPeerID {
			continue
		}

		pi.peerID = newPeerID
		pi.addr = utils.CopyUDPAddr(newAddr)
		pi.nextSend = time.Now()

		//если сообщение было адресовано старому мастеру — перепишем receiver_id
		if pi.msg != nil && pi.msg.GetReceiverId() == oldPeerID {
			pi.msg.ReceiverId = proto.Int32(newPeerID)
		}
	}
}
