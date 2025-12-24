package client

import (
	"log/slog"
	"net"
	"snake-game/internal/application/utils"
	snakespb "snake-game/internal/infrastructure/network/proto"
	"time"
)

// sendJob — задача на отправку одного UDP сообщения
//
//	-> отправляем один раз
type sendJob struct {
	msg           *snakespb.GameMessage
	addr          *net.UDPAddr
	peerID        int32
	retryInterval time.Duration
	maxAttempts   int
	reliable      bool
}

// startSendWorkers — запускает N воркеров, которые читают очередь sendQ и шлют UDP
func (c *Client) startSendWorkers(n int) {
	for i := 0; i < n; i++ {
		go func(workerID int) {
			for {
				select {
				case job := <-c.sendQ:
					if job.addr == nil || job.msg == nil {
						continue
					}

					if job.reliable {
						_, _, _ = c.protectedSend(job.msg, job.addr, job.peerID, job.retryInterval, job.maxAttempts)
					} else {
						// “быстрая” отправка без ожидания Ack для Announcement и Discover
						if err := c.clientTransport.SendTo(job.msg, job.addr); err == nil && job.peerID != 0 {
							c.markSent(job.peerID)
						}
					}

				case <-c.sendStop:
					return
				}
			}
		}(i)
	}
}

// enqueueProtected — кладём надёжную отправку в очередь sendQ
func (c *Client) enqueueProtected(msg *snakespb.GameMessage, addr *net.UDPAddr, peerID int32, retry time.Duration, attempts int) {
	select {
	case c.sendQ <- sendJob{
		msg: msg, addr: addr, peerID: peerID,
		retryInterval: retry, maxAttempts: attempts,
		reliable: true,
	}:
	default:
		// очередь переполнена — лучше дропнуть, чем плодить горутины
		slog.Info("sendQ overflow: drop reliable", "msg type", utils.MsgTypeName(msg))
	}
}

// enqueueNotProtected — кладём быструю отправку в очередь sendQ (без Ack)
// используется для ping, первичной отправки pending и т.п.
func (c *Client) enqueueNotProtected(msg *snakespb.GameMessage, addr *net.UDPAddr, peerID int32) {
	select {
	case c.sendQ <- sendJob{msg: msg, addr: addr, peerID: peerID, reliable: false}:
	default:
		slog.Info("sendQ overflow: drop ", "msg type", utils.MsgTypeName(msg))
	}
}
