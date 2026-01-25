package client

import (
	"log/slog"
	"net"
	snakespb "snake-game/internal/infrastructure/network/proto"

	"google.golang.org/protobuf/proto"
)

func (c *Client) buildErrorMessage(seq int64, reason string) *snakespb.GameMessage {
	return &snakespb.GameMessage{
		MsgSeq: proto.Int64(seq),
		Type: &snakespb.GameMessage_Error{
			Error: &snakespb.GameMessage_ErrorMsg{
				ErrorMessage: proto.String(reason),
			},
		},
	}
}

func (c *Client) handleError(
	msg *snakespb.GameMessage,
	m *snakespb.GameMessage_Error,
	addr *net.UDPAddr,
) {
	errText := m.Error.GetErrorMessage()
	if errText == "" {
		errText = "Неизвестная ошибка от узла " + addr.String()
	}

	slog.Error("ERROR",
		"from", addr, "seq", msg.GetMsgSeq(), "msg", errText)

	if c.view != nil {
		c.view.ShowError(errText)
	}
}
