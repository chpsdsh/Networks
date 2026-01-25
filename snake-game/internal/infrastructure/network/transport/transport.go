package transport

import (
	"fmt"
	"log/slog"
	"net"
	"snake-game/internal/application/client"
	"snake-game/internal/application/utils"
	snakespb "snake-game/internal/infrastructure/network/proto"

	"google.golang.org/protobuf/proto"
)

// Transport — отвечает за UDP unicast + UDP multicast
type Transport struct {
	mcGroup string
	mcPort  int

	multicastConn *net.UDPConn // только приём multicast
	generalConn   *net.UDPConn // отправка unicast и multicast

	handler client.Handler
}

// Создается два сокета один для unicast второй для multicast
func NewTransport(h client.Handler, mcGroup string, mcPort int) (*Transport, error) {

	t := &Transport{
		mcGroup: mcGroup,
		mcPort:  mcPort,
		handler: h,
	}

	var err error
	t.multicastConn, err = createMulticastListener(mcGroup, mcPort)
	if err != nil {
		return nil, fmt.Errorf("multicast listen: %w", err)
	}

	t.generalConn, err = createUnicastListener()
	if err != nil {
		return nil, fmt.Errorf("unicast listen: %w", err)
	}

	go t.readMulticastLoop()
	go t.readUnicastLoop()

	return t, nil
}

// сокет 1 — только приём multicast
func createMulticastListener(group string, port int) (*net.UDPConn, error) {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(group),
		Port: port,
	}

	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		return nil, err
	}
	if err := conn.SetReadBuffer(64 * 1024); err != nil {
		return nil, err
	}
	return conn, nil
}

// сокет 2 — обычный UDP на порту 0
func createUnicastListener() (*net.UDPConn, error) {
	addr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 0,
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, err
	}
	if err := conn.SetReadBuffer(64 * 1024); err != nil {
		return nil, err
	}
	return conn, nil
}

// readMulticastLoop — бесконечно читаем multicast и передаём в общий обработчик
func (t *Transport) readMulticastLoop() {
	buf := make([]byte, 64*1024)
	for {
		n, src, err := t.multicastConn.ReadFromUDP(buf)
		if err != nil {

			return
		}
		t.handleRawPacket(buf[:n], src)
	}
}

// readUnicastLoop — бесконечно читаем unicast и передаём в общий обработчик
func (t *Transport) readUnicastLoop() {
	buf := make([]byte, 64*1024)
	for {
		n, src, err := t.generalConn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		t.handleRawPacket(buf[:n], src)
	}
}

// handleRawPacket — общий разбор пакета: self-check -> proto.Unmarshal -> handler.HandlePacket
func (t *Transport) handleRawPacket(data []byte, src *net.UDPAddr) {
	if t.handler == nil {
		return
	}

	// защита от “эхо”
	if t.IsSelfPacketUnicast(src) {
		slog.Info("RECV self-packet — ignored", "from", src)
		return
	}

	// декодируем protobuf GameMessage
	var msg snakespb.GameMessage
	if err := proto.Unmarshal(data, &msg); err != nil {
		slog.Error("RECV", "from=%s", src, "unmarshall error", err)
		return
	}

	slog.Info("RECV ",
		"from", src, "bytes", len(data), "seq", msg.GetMsgSeq(), "type", utils.MsgTypeName(&msg))
	t.handler.HandlePacket(&msg, src)
}

func (t *Transport) IsSelfPacketUnicast(src *net.UDPAddr) bool {
	if t.generalConn == nil || src == nil {
		return false
	}

	local := t.generalConn.LocalAddr().(*net.UDPAddr)

	if local.IP.IsUnspecified() {
		return src.Port == local.Port
	}

	return src.IP.Equal(local.IP) && src.Port == local.Port
}

// SendTo — отправка unicast UDP пакета
func (t *Transport) SendTo(msg *snakespb.GameMessage, addr *net.UDPAddr) error {
	data, err := proto.Marshal(msg)
	if err != nil {

		return err
	}
	n, err := t.generalConn.WriteToUDP(data, addr)
	if err != nil {
		slog.Error("SEND",
			"to", addr, "seq", msg.GetMsgSeq(), "type", utils.MsgTypeName(msg), "err", err)
		return err
	}

	slog.Info("SEND",
		"to", addr, "bytes", n, "seq", msg.GetMsgSeq(), "type", utils.MsgTypeName(msg))
	return nil
}

// SendToMulticast — отправка Announce и Discover на multicast-группу
func (t *Transport) SendToMulticast(msg *snakespb.GameMessage) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	dst := &net.UDPAddr{
		IP:   net.ParseIP(t.mcGroup),
		Port: t.mcPort,
	}
	n, err := t.generalConn.WriteToUDP(data, dst)
	if err != nil {
		slog.Error("SEND-MC ",
			"to", dst, "seq", msg.GetMsgSeq(), "type", utils.MsgTypeName(msg), "error", err)
		return err
	}
	slog.Info("SEND-MC ",
		"to", dst, "bytes", n, "seq", msg.GetMsgSeq(), "type", utils.MsgTypeName(msg))
	slog.Info("msg")

	return nil
}

// LocalAddr — чтобы знать, на каком порту сидит наш unicast-сокет
func (t *Transport) LocalAddr() *net.UDPAddr {
	if t.generalConn == nil {
		return nil
	}
	return t.generalConn.LocalAddr().(*net.UDPAddr)
}
