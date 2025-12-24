package client

import (
	"net"
	"snake-game/internal/application/game"
	"snake-game/internal/domain"
	snakespb "snake-game/internal/infrastructure/network/proto"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

type View interface {
	ShowStartMenu()
	ShowError(msg string)
	ShowConfigMenu()
	ShowGameScreen(game *game.Game)
	RefreshBoard()
	ShowGames()
	RefreshGamesList()
	RefreshRating()
	RefreshAvailableGames()
}

// Handler — куда транспорт отдаёт уже распарсенные GameMessage
type Handler interface {
	HandlePacket(msg *snakespb.GameMessage, addr *net.UDPAddr)
}

const SendWorkersNum = 4

type Client struct {
	view            View             // GUI
	game            *game.Game       // локальное состояние игры
	node            domain.NodeState // моя роль/ID + кто master/deputy
	clientTransport domain.Transport // UDP транспорт (unicast + multicast)

	//Адреса игроков и тайминги для мониторинга
	playersAddr   map[int32]*net.UDPAddr // playerID -> UDP адрес
	lastHeard     map[int32]time.Time    // когда последний раз получили сообщение от игрока (для timeout)
	lastSent      map[int32]time.Time    // когда последний раз отправили сообщение игроку (ping)
	playerAddrsMu sync.RWMutex

	//stop-каналы фоновых циклов
	tickerStop   chan struct{} // стоп игрового тика master’а (OneTick + broadcastState)
	announceStop chan struct{} // стоп анонсов (multicast announcement)
	monitorStop  chan struct{} // стоп мониторинга (ping + timeout / failover)

	//MsgSeq для протокола (уникальные номера сообщений)
	seq   int64
	seqMu sync.Mutex

	//Список обнаруженных игр
	gamesMu  sync.RWMutex
	games    map[string]*domain.DiscoveredGame
	GameName string

	//Надёжная доставка (pending + ack)
	pendingMu sync.Mutex
	pending   map[int64]*pendingItem // msg_seq -> ожидание Ack/ретраи

	//Порядок состояний (StateOrder), чтобы клиенты не принимали старые состояния
	stateOrder     int32 // счётчик stateOrder у мастера
	stateOrderMu   sync.Mutex
	lastStateOrder int32 // последний принятый stateOrder на клиенте

	// Очередь отправки
	sendQ    chan sendJob  // очередь сообщений на отправку
	sendStop chan struct{} // стоп воркеров отправки

	//Steer: буфер поворотов на тик
	steerMu      sync.Mutex
	pendingSteer map[int32]domain.Direction // накопленные повороты на следующий тик
	lastSteerSeq map[int32]int64            // защита от старых steer

	//Фоновый цикл ретраев pending сообщений
	reliabilityStop chan struct{}

	//Флаг “мы сейчас в игре” (чтобы игнорить пакеты вне игры)
	inGameMu sync.RWMutex
	inGame   bool
}

// NewGameClient — создаём клиент и запускаем воркеры отправки
func NewGameClient() *Client {

	c := &Client{
		games:   make(map[string]*domain.DiscoveredGame),
		pending: make(map[int64]*pendingItem),

		sendQ:    make(chan sendJob, 1024),
		sendStop: make(chan struct{}),
	}

	// фиксированное число воркеров, которые читают sendQ и шлют UDP
	c.startSendWorkers(SendWorkersNum)

	return c
}

func (c *Client) GetRole() snakespb.NodeRole {
	return c.node.SelfRole
}

func (c *Client) SetView(view View) { c.view = view }

func (c *Client) SetTransport(t domain.Transport) { c.clientTransport = t }

func (c *Client) setInGame(v bool) {
	c.inGameMu.Lock()
	c.inGame = v
	c.inGameMu.Unlock()
}

func (c *Client) isInGame() bool {
	c.inGameMu.RLock()
	defer c.inGameMu.RUnlock()
	return c.inGame
}

func (c *Client) ChangeDirection(dir domain.Direction) {
	if c.game == nil || !c.isInGame() {
		return
	}

	switch c.node.SelfRole {
	case snakespb.NodeRole_MASTER:

		c.game.SetDirection(dir, c.node.SelfID)

	case snakespb.NodeRole_NORMAL, snakespb.NodeRole_DEPUTY:

		if c.node.MasterAddr == nil || c.node.MasterID == 0 {
			return
		}

		msg := c.buildSteerMessage(dir)
		if msg == nil {
			return
		}

		msg.ReceiverId = proto.Int32(c.node.MasterID)

		// ретраи state_delay/10
		retry := time.Duration(c.game.Config().StateDelayMs/10) * time.Millisecond
		if retry <= 0 {
			retry = 100 * time.Millisecond
		}

		// отправляем с ожиданием Ack (через pending + reliability loop)
		_, _, _ = c.protectedSend(msg, c.node.MasterAddr, c.node.MasterID, retry, 3)

	default:
		// viewer не управляет
	}
}
