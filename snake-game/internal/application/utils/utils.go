package utils

import (
	"fmt"
	"net"
	"snake-game/internal/domain"
	snakespb "snake-game/internal/infrastructure/network/proto"
	"time"

	"google.golang.org/protobuf/proto"
)

func MsgTypeName(msg *snakespb.GameMessage) string {
	switch msg.Type.(type) {
	case *snakespb.GameMessage_Ping:
		return "Ping"
	case *snakespb.GameMessage_Steer:
		return "Steer"
	case *snakespb.GameMessage_Ack:
		return "Ack"
	case *snakespb.GameMessage_State:
		return "State"
	case *snakespb.GameMessage_Announcement:
		return "Announcement"
	case *snakespb.GameMessage_Join:
		return "Join"
	case *snakespb.GameMessage_Error:
		return "Error"
	case *snakespb.GameMessage_RoleChange:
		return "RoleChange"
	case *snakespb.GameMessage_Discover:
		return "Discover"
	default:
		return "Unknown"
	}
}

func CopyUDPAddr(a *net.UDPAddr) *net.UDPAddr {
	if a == nil {
		return nil
	}
	cp := *a
	return &cp
}

// generateGameName — уникальное имя игры
func GenerateGameName(playerName string) string {
	return fmt.Sprintf("%s-%d", playerName, time.Now().UnixNano())
}

// запрет подключаться в игру, в которой ты уже есть
func SameAddr(a, b *net.UDPAddr) bool {
	if a == nil || b == nil {
		return false
	}
	return a.IP.Equal(b.IP) && a.Port == b.Port
}

func BuildSnakeState(s *domain.Snake) *snakespb.GameState_Snake {
	points := make([]*snakespb.GameState_Coord, 0, len(s.Body))

	// голова — абсолют
	head := s.Body[0]
	points = append(points, &snakespb.GameState_Coord{
		X: proto.Int32(int32(head.X)),
		Y: proto.Int32(int32(head.Y)),
	})

	// остальные — смещения
	for i := 1; i < len(s.Body); i++ {
		prev := s.Body[i-1]
		cur := s.Body[i]
		points = append(points, &snakespb.GameState_Coord{
			X: proto.Int32(int32(cur.X - prev.X)),
			Y: proto.Int32(int32(cur.Y - prev.Y)),
		})
	}

	state := snakespb.GameState_Snake_ALIVE
	if s.State == domain.SnakeZombie {
		state = snakespb.GameState_Snake_ZOMBIE
	}

	dir := toProtoDirection(s.Direction)

	return &snakespb.GameState_Snake{
		PlayerId:      proto.Int32(s.PlayerID),
		Points:        points,
		State:         &state,
		HeadDirection: &dir,
	}
}

func toProtoDirection(d domain.Direction) snakespb.Direction {
	switch d {
	case domain.DirUp:
		return snakespb.Direction_UP
	case domain.DirDown:
		return snakespb.Direction_DOWN
	case domain.DirLeft:
		return snakespb.Direction_LEFT
	case domain.DirRight:
		return snakespb.Direction_RIGHT
	default:
		return snakespb.Direction_UP
	}
}

func DecodePbSnake(ps *snakespb.GameState_Snake) *domain.Snake {
	pts := ps.GetPoints()
	if len(pts) == 0 {
		return &domain.Snake{PlayerID: ps.GetPlayerId()}
	}

	body := make([]domain.Coords, 0, len(pts))

	// первая точка — абсолютная
	head := pts[0]
	x := int(head.GetX())
	y := int(head.GetY())
	body = append(body, domain.Coords{X: x, Y: y})

	// остальные — смещения
	for i := 1; i < len(pts); i++ {
		dx := int(pts[i].GetX())
		dy := int(pts[i].GetY())
		x += dx
		y += dy
		body = append(body, domain.Coords{X: x, Y: y})
	}

	// правильное направление
	dir, ok := FromProtoDirection(ps.GetHeadDirection())
	if !ok {
		dir = domain.DirUp
	}

	state := domain.SnakeAlive
	if ps.GetState() == snakespb.GameState_Snake_ZOMBIE {
		state = domain.SnakeZombie
	}

	return &domain.Snake{
		PlayerID:  ps.GetPlayerId(),
		Body:      body,
		Direction: dir,
		State:     state,
	}
}

func FromProtoDirection(d snakespb.Direction) (domain.Direction, bool) {
	switch d {
	case snakespb.Direction_UP:
		return domain.DirUp, true
	case snakespb.Direction_DOWN:
		return domain.DirDown, true
	case snakespb.Direction_LEFT:
		return domain.DirLeft, true
	case snakespb.Direction_RIGHT:
		return domain.DirRight, true
	default:
		return 0, false
	}
}
