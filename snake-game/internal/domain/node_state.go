package domain

import (
	"net"
	snakespb "snake-game/internal/infrastructure/network/proto"
)

type NodeState struct {
	SelfID   int32
	SelfRole snakespb.NodeRole

	MasterID   int32
	MasterAddr *net.UDPAddr

	DeputyID   int32
	DeputyAddr *net.UDPAddr
}
