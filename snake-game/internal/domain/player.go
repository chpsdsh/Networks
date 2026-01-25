package domain

import snakespb "snake-game/internal/infrastructure/network/proto"

type Player struct {
	ID    int32
	Name  string
	Score int32
	Role  snakespb.NodeRole
}

type PlayerInfo struct {
	ID    int32
	Name  string
	Score int32
}
