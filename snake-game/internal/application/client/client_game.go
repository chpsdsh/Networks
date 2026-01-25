package client

import (
	"fmt"
	"log/slog"
	"snake-game/internal/domain"
	snakespb "snake-game/internal/infrastructure/network/proto"
	"time"
)

func (c *Client) StopGame() {
	if c.tickerStop != nil {
		close(c.tickerStop)
		c.tickerStop = nil
	}
	if c.monitorStop != nil {
		close(c.monitorStop)
		c.monitorStop = nil
	}
	if c.reliabilityStop != nil {
		close(c.reliabilityStop)
		c.reliabilityStop = nil
	}

	c.stopAnnouncement()
	c.stopSendWorkers()

	// чтобы после выхода не летели старые ретраи/ping’и
	c.clearAllPending(fmt.Errorf("game stopped"))

	c.game = nil

	c.setInGame(false)
}

func (c *Client) stopSendWorkers() {
	if c.sendStop == nil {
		return
	}
	select {
	case <-c.sendStop:
		// уже закрыт
	default:
		close(c.sendStop)
	}
	c.sendStop = nil
}

func (c *Client) ensureSendWorkers() {
	if c.sendStop != nil {
		return // уже есть живые воркеры
	}
	c.sendStop = make(chan struct{})
	c.startSendWorkers(4)
}

func (c *Client) makeSnakeZombie(id int32) {
	if c.game == nil {
		return
	}

	// Змея становится зомби, если есть
	if s, ok := c.game.Snakes[id]; ok && s != nil {
		s.State = domain.SnakeZombie
	}

	// Игрок перестаёт быть NORMAL, становится VIEWER
	if p, ok := c.game.Players[id]; ok && p != nil {
		p.Role = snakespb.NodeRole_VIEWER
	}
}

func (c *Client) StartGame() {
	// остановить старый цикл если был
	if c.tickerStop != nil {
		close(c.tickerStop)
	}
	fmt.Println(c.game.Tick)
	c.view.ShowGameScreen(c.game)

	c.tickerStop = make(chan struct{})
	delay := time.Millisecond * time.Duration(c.game.Tick)

	if delay <= 0 {
		delay = 200 * time.Millisecond
	}

	slog.Info("StartGame:", "delay", delay, "snakes", len(c.game.Snakes))
	go func() {
		ticker := time.NewTicker(time.Millisecond * time.Duration(c.game.Tick))
		defer ticker.Stop()

		c.startMonitorLoop()
		for {
			select {
			case <-ticker.C:
				if c.game == nil {
					continue
				}
				c.applyPendingSteer()
				c.game.MoveSnakes()
				c.broadcastState()
				c.view.RefreshBoard()
				c.view.RefreshRating()

			case <-c.tickerStop:
				return
			}
		}
	}()
}
