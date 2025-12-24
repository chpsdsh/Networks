package domain

import (
	"net"
	"time"
)

type DiscoveredGame struct {
	GameName string
	Host     string // ip:port
	Players  int
	Width    int
	Height   int
	CanJoin  bool

	FoodStatic   int
	StateDelayMs int

	Addr     *net.UDPAddr // куда слать JoinMsg
	LastSeen time.Time
}
