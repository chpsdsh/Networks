package domain

import (
	"net"
	snakespb "snake-game/internal/infrastructure/network/proto"
)

type Transport interface {
	SendTo(msg *snakespb.GameMessage, addr *net.UDPAddr) error
	SendToMulticast(msg *snakespb.GameMessage) error
	LocalAddr() *net.UDPAddr
}
