package client

import (
	"fmt"
	"log/slog"
	"snake-game/internal/application/game"
	"snake-game/internal/application/utils"
	"snake-game/internal/domain"
	snakespb "snake-game/internal/infrastructure/network/proto"
	"sort"
	"time"

	"google.golang.org/protobuf/proto"
)

// Run — старт клиента: показываем главное меню
func (c *Client) Run() {
	c.view.ShowStartMenu()
}

// CreateGame — переход к экрану создания новой игры
func (c *Client) CreateGame() {
	c.view.ShowConfigMenu()
}

// NewGameFromInGame — начать новую игру, находясь уже в игре
func (c *Client) NewGameFromInGame() {
	c.StopGame()
	c.CreateGame()
}

// ShowGameList — запросить список игр по multicast и показать их
func (c *Client) ShowGameList() {
	if c.clientTransport != nil {
		// Discover-сообщение для поиска активных игр
		msg := &snakespb.GameMessage{
			MsgSeq: proto.Int64(c.nextSeq()),
			Type: &snakespb.GameMessage_Discover{
				Discover: &snakespb.GameMessage_DiscoverMsg{},
			},
		}
		_ = c.clientTransport.SendToMulticast(msg)
	}

	c.view.ShowGames()
}

// BackToStart — полный выход в стартовое меню
func (c *Client) BackToStart() {
	c.fullShutdown()
	c.view.ShowStartMenu()
}

// LeaveGame — выход из текущей игры (для кнопки "Выход")
func (c *Client) LeaveGame() {
	c.fullShutdown()
	if c.view != nil {
		c.view.ShowStartMenu()
	}
}

// fullShutdown — полный сброс состояния клиента и сети
func (c *Client) fullShutdown() {
	c.StopGame()

	// перестаём быть игровым узлом
	c.node = domain.NodeState{}

	// очищаем данные о других игроках
	c.playerAddrsMu.Lock()
	c.playersAddr = nil
	c.lastHeard = nil
	c.lastSent = nil
	c.playerAddrsMu.Unlock()
}

// CreateNewGame — создание новой игры и становление MASTER
func (c *Client) CreateNewGame(playerName string, cfg domain.GameConfig) error {
	// создаём доменную игру
	game := game.NewGame(cfg)

	// добавляем первого игрока (master)
	id := game.AddFirstPlayer(playerName)

	c.game = game
	c.ensureSendWorkers()
	c.setInGame(true)

	// инициализация состояния узла
	c.GameName = utils.GenerateGameName(playerName)
	c.node.SelfID = id
	c.node.SelfRole = snakespb.NodeRole_MASTER
	c.node.MasterID = id

	// создаём змейку мастера
	if err := c.game.SpawnSnake(id); err != nil {
		return fmt.Errorf("spawn snake: %w", err)
	}

	// начальная еда
	c.game.InitFood()

	// мастер знает свой сетевой адрес
	if c.clientTransport != nil {
		c.node.MasterAddr = c.clientTransport.LocalAddr()
	}

	// запуск игры и сервисных циклов
	c.StartGame()
	c.startAnnouncement()
	c.startReliabilityLoop(c.reliabilityInterval())

	slog.Info("CreateNewGame: ", "name",
		playerName, "gameName", c.GameName, "cfg", cfg)
	return nil
}

func (c *Client) gamesTTL() time.Duration {
	if c.isInGame() {
		return 2 * time.Second // внутри игры — только реально живые анонсы
	}
	return 5 * time.Second // в меню можно дольше
}

func (c *Client) pruneGamesLocked(now time.Time) {
	ttl := c.gamesTTL()
	for k, g := range c.games {
		if g == nil || now.Sub(g.LastSeen) > ttl {
			delete(c.games, k)
		}
	}
}

// GamesSnapshot — актуальный список доступных игр (для UI)
func (c *Client) GamesSnapshot() []domain.DiscoveredGame {
	c.gamesMu.RLock()
	defer c.gamesMu.RUnlock()

	now := time.Now()
	c.pruneGamesLocked(now)
	out := make([]domain.DiscoveredGame, 0, len(c.games))

	for _, g := range c.games {
		if g == nil {
			continue
		}

		ttl := c.gamesTTL()

		if now.Sub(g.LastSeen) > ttl {
			continue
		}

		// 1) не показываем игру, в которой мы сейчас находимся
		if c.isInGame() && g.GameName == c.GameName {
			continue
		}

		// 2) показываем только то, куда можно подключиться (опционально)
		if c.isInGame() && !g.CanJoin {
			continue
		}

		out = append(out, *g)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].GameName == out[j].GameName {
			return out[i].Host < out[j].Host
		}
		return out[i].GameName < out[j].GameName
	})

	return out
}
