package application

import (
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"socks-proxy/internal/domain"
	"socks-proxy/internal/infrastructure/dnsclient"
	"socks-proxy/internal/infrastructure/netpoll"
	"socks-proxy/internal/infrastructure/network"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	MaxEpollEvents = 128
	DnsClientPort  = 53
)

type Server struct {
	port   int
	poller *netpoll.Poller

	listenerFd  int
	dnsFd       int
	connManager *ConnectionManager
	dnsClient   *dnsclient.Client
}

func NewServer(port int) (*Server, error) {
	poller, err := netpoll.NewPoller()
	if err != nil {
		return nil, err
	}

	listenerFd, err := network.CreateTcpListener(port)
	if err != nil {
		_ = poller.Close()
		return nil, err
	}

	dnsFd, err := network.CreateUdpSocket()
	if err != nil {
		_ = network.Close(listenerFd)
		_ = poller.Close()
		return nil, err
	}

	if err := poller.Add(listenerFd, netpoll.EventRead); err != nil {
		_ = poller.Close()
		_ = network.Close(listenerFd)
		_ = network.Close(dnsFd)
		return nil, err
	}

	if err := poller.Add(dnsFd, netpoll.EventRead); err != nil {
		_ = poller.Close()
		_ = network.Close(dnsFd)
		_ = network.Close(listenerFd)
		return nil, err
	}

	resolverIP := [4]byte{8, 8, 8, 8}
	dnsClient := dnsclient.NewClient(resolverIP, DnsClientPort)

	return &Server{
		port:        port,
		poller:      poller,
		listenerFd:  listenerFd,
		dnsFd:       dnsFd,
		connManager: NewConnectionManager(),
		dnsClient:   dnsClient,
	}, nil
}

func (s *Server) failConnection(conn *domain.Connection, reason string, conErr error) {
	slog.Error("connection failed",
		"connID", conn.ID,
		"reason", reason,
		"error", conErr,
	)

	resp := []byte{
		domain.SocksVersion5,
		0x01,
		0x00,
		0x01,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00,
	}

	_ = s.poller.Del(conn.ClientFD)
	if conn.TargetFD >= 0 {
		_ = s.poller.Del(conn.TargetFD)
	}

	_, _ = network.Write(conn.ClientFD, resp)

	_ = network.Close(conn.ClientFD)

	if conn.TargetFD >= 0 {
		_ = network.Close(conn.TargetFD)
	}

	s.connManager.DeleteConnection(conn)

}

func (s *Server) Start() error {
	slog.Info("SOCKS5 proxy listening on 0.0.0.0", "port", s.port)
	for {
		events, err := s.poller.Wait(MaxEpollEvents, -1)
		if err != nil {
			return err
		}
		for _, ev := range events {
			fd := ev.FD
			mask := ev.Events
			switch {
			case fd == s.listenerFd && mask&netpoll.EventRead != 0:
				s.handleAccept()
			case fd == s.dnsFd && mask&netpoll.EventRead != 0:
				s.handleDnsReadable()
			default:
				s.handleFd(fd, mask)
			}
		}
	}
}

func (s *Server) handleAccept() {
	for {
		clientFd, _, err := network.Accept(s.listenerFd)
		if err != nil {
			if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) {
				return
			}
			slog.Error("SOCKS5 proxy accept error:", err)
			return
		}

		if err := s.poller.Add(clientFd, netpoll.EventRead|netpoll.EventError|netpoll.EventHup); err != nil {
			slog.Error("poller add client error:", err)
			_ = network.Close(clientFd)
			return
		}

		conn := s.connManager.NewClientConnection(clientFd)
		slog.Info("new client", "fd", clientFd, "connection", conn.ID)
	}
}

func (s *Server) handleDnsReadable() {
	buf := make([]byte, 1500)

	n, err := network.RecvFromIPv4(s.dnsFd, buf)
	if err != nil {
		if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) {
			return
		}
		slog.Error("dns recv error", "err", err)
		return
	}

	queryID, ip4, err := s.dnsClient.ParseResponse(buf[:n])
	if err != nil {
		slog.Error("dns parse error", "err", err)
		return
	}

	conn := s.connManager.GetDnsConnectionById(queryID)
	if conn == nil {
		return
	}

	targetFD, err := network.ConnectTcp4(ip4, int(conn.PendingPort))
	if err != nil {
		s.failConnection(conn, "connect after DNS", err)
		return
	}

	s.connManager.AttachTarget(conn, targetFD)

	_ = s.poller.Add(targetFD, netpoll.EventWrite|netpoll.EventError|netpoll.EventHup)

	conn.State = domain.StateConnectingTarget

}

func (s *Server) handleFd(fd int, mask netpoll.EventMask) {
	if conn := s.connManager.GetClientConnectionByFd(fd); conn != nil {
		s.handleClientEvent(conn, mask)
		return
	}

	if conn := s.connManager.GetTargetConnectionByFd(fd); conn != nil {
		s.handleTargetEvent(conn, mask)
		return
	}

	slog.Error("unknown fd and mask:", "fd", fd, "mask", mask)
}

func (s *Server) handleClientEvent(conn *domain.Connection, mask netpoll.EventMask) {
	if mask&netpoll.EventError != 0 {
		soErr, _ := unix.GetsockoptInt(conn.ClientFD, unix.SOL_SOCKET, unix.SO_ERROR)
		if soErr != 0 {
			err := syscall.Errno(soErr)
			s.failConnection(conn, "client error", err)
		} else {
			s.failConnection(conn, "client error", errors.New("EPOLLERR but SO_ERROR=0"))
		}
		return
	}

	if mask&netpoll.EventRead != 0 {
		if err := s.handleClientReadable(conn); err != nil {
			s.failConnection(conn, "client read", err)
			return
		}
	}

	if mask&netpoll.EventWrite != 0 {
		if err := s.handleClientWritable(conn); err != nil {
			s.failConnection(conn, "client write", err)
			return
		}
	}
}

func (s *Server) handleTargetEvent(conn *domain.Connection, mask netpoll.EventMask) {

	if mask&netpoll.EventError != 0 {
		soErr, _ := unix.GetsockoptInt(conn.TargetFD, unix.SOL_SOCKET, unix.SO_ERROR)
		if soErr != 0 {
			err := syscall.Errno(soErr)
			s.failConnection(conn, "target error", err)
		} else {
			s.failConnection(conn, "target error", errors.New("EPOLLERR but SO_ERROR=0"))
		}
		return
	}

	if conn.State == domain.StateConnectingTarget && mask&netpoll.EventWrite != 0 {
		if err := s.finishTargetConnect(conn); err != nil {
			s.failConnection(conn, "finish target connect", err)
			return
		}
	}

	if conn.State == domain.StateRelaying {
		if mask&netpoll.EventRead != 0 {
			if err := s.handleTargetReadable(conn); err != nil {
				s.failConnection(conn, "target read", err)
				return
			}
		}

		if mask&netpoll.EventWrite != 0 {
			if err := s.handleTargetWritable(conn); err != nil {
				s.failConnection(conn, "target write", err)
				return
			}
		}
	}
}

func (s *Server) handleClientReadable(conn *domain.Connection) error {
	buf := make([]byte, 4096)

	n, err := network.Read(conn.ClientFD, buf)
	if n > 0 {
		conn.ToTargetBuf = append(conn.ToTargetBuf, buf[:n]...)
	}

	if err != nil {
		if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) {
			return nil
		}
		if errors.Is(err, io.EOF) && n == 0 {
			_ = unix.Shutdown(conn.ClientFD, unix.SHUT_RD)
			_ = unix.Shutdown(conn.TargetFD, unix.SHUT_WR)
			conn.ClientReadClosed = true
			conn.TargetWriteClosed = true
			if conn.ClientWriteClosed && conn.TargetReadClosed {
				s.closeConnection(conn)
			}
			return nil
		}
		return err
	}

	if n == 0 {
		_ = unix.Shutdown(conn.ClientFD, unix.SHUT_RD)
		_ = unix.Shutdown(conn.TargetFD, unix.SHUT_WR)
		conn.ClientReadClosed = true
		conn.TargetWriteClosed = true
		if conn.ClientWriteClosed && conn.TargetReadClosed {
			s.closeConnection(conn)
		}
		return nil
	}

	switch conn.State {
	case domain.StateGreeting:
		done, err := s.processGreeting(conn)
		if err != nil {
			return err
		}
		if !done {
			return nil
		}

	case domain.StateRequest:
		done, err := s.processRequest(conn)
		if err != nil {
			return err
		}
		if !done {
			return nil
		}

	case domain.StateRelaying:
		if conn.TargetFD >= 0 && len(conn.ToTargetBuf) > 0 {
			return s.flushToTarget(conn)
		}
		return nil

	default:
		return nil
	}

	return nil
}

func (s *Server) processGreeting(conn *domain.Connection) (bool, error) {
	buf := conn.ToTargetBuf
	if len(buf) < 2 {
		return false, nil
	}

	ver := buf[0]
	nMethods := int(buf[1])

	if ver != domain.SocksVersion5 {
		return false, errors.New("unsupported SOCKS version")
	}

	if len(buf) < 2+nMethods {
		return false, nil
	}

	methods := buf[2 : 2+nMethods]

	supportsNoAuth := false
	for _, m := range methods {
		if m == domain.SocksNoAuth {
			supportsNoAuth = true
			break
		}
	}
	if !supportsNoAuth {
		resp := []byte{domain.SocksVersion5, domain.SocksNoAccessibleAuthMethod}
		_, _ = network.Write(conn.ClientFD, resp)
		return false, errors.New("no acceptable auth methods")
	}

	resp := []byte{domain.SocksVersion5, domain.SocksNoAuth}
	if _, err := network.Write(conn.ClientFD, resp); err != nil {
		return false, err
	}

	conn.ToTargetBuf = buf[2+nMethods:]

	conn.State = domain.StateRequest

	return true, nil
}

func (s *Server) processRequest(conn *domain.Connection) (bool, error) {
	buf := conn.ToTargetBuf
	if len(buf) < 4 {
		return false, nil
	}

	ver := buf[0]
	cmd := buf[1]
	addrType := buf[3]

	if ver != domain.SocksVersion5 {
		return false, errors.New("invalid VER in request")
	}

	if cmd != domain.SocksEstablishTcpConnection {
		resp := []byte{domain.SocksVersion5, domain.SocksCommandNotSupported, 0x00, 0x01, 0, 0, 0, 0, 0, 0}
		_, _ = network.Write(conn.ClientFD, resp)
		return false, errors.New("only CONNECT is supported")
	}

	offset := 4

	var ip4 [4]byte
	var dom string
	var port uint16

	switch addrType {
	case domain.SocksAddrTypeIPv4:
		if len(buf) < offset+4+2 {
			return false, nil
		}
		copy(ip4[:], buf[offset:offset+4])
		offset += 4

		port = binary.BigEndian.Uint16(buf[offset : offset+2])
		offset += 2

		conn.PendingPort = port
		if err := s.startConnectIPv4(conn, ip4, port); err != nil {
			return false, err
		}

	case domain.SocksAddrTypeDomain:
		if len(buf) < offset+1 {
			return false, nil
		}
		nameLen := int(buf[offset])
		offset++

		if len(buf) < offset+nameLen+2 {
			return false, nil
		}
		dom = string(buf[offset : offset+nameLen])
		offset += nameLen

		port = binary.BigEndian.Uint16(buf[offset : offset+2])
		offset += 2

		if err := s.startDNSResolve(conn, dom, port); err != nil {
			return false, err
		}

	case domain.SocksAddrTypeIPv6:
		resp := []byte{domain.SocksVersion5, domain.SocksAddrTypeNotSupported, 0x00, 0x01, 0, 0, 0, 0, 0, 0}
		_, _ = network.Write(conn.ClientFD, resp)
		return false, errors.New("IPv6 not supported")
	default:
		resp := []byte{domain.SocksVersion5, domain.SocksAddrTypeNotSupported, 0x00, 0x01, 0, 0, 0, 0, 0, 0}
		_, _ = network.Write(conn.ClientFD, resp)
		return false, errors.New("unknown address type")
	}

	conn.ToTargetBuf = buf[offset:]

	return true, nil
}

func (s *Server) startConnectIPv4(conn *domain.Connection, ip [4]byte, port uint16) error {
	targetFD, err := network.ConnectTcp4(ip, int(port))
	if err != nil {
		return err
	}

	s.connManager.AttachTarget(conn, targetFD)

	if err := s.poller.Add(targetFD, netpoll.EventWrite|netpoll.EventError|netpoll.EventHup); err != nil {
		_ = network.Close(targetFD)
		return err
	}

	conn.State = domain.StateConnectingTarget
	return nil
}

func (s *Server) startDNSResolve(conn *domain.Connection, dom string, port uint16) error {
	queryID, packet, err := s.dnsClient.BuildQuery(dom)
	if err != nil {
		return err
	}

	conn.PendingDomain = dom
	conn.PendingPort = port
	s.connManager.BindDNSQuery(conn, queryID)

	if err := network.SendToIPv4(s.dnsFd, s.dnsClient.ResolverIp, s.dnsClient.ResolverPort, packet); err != nil {
		return err
	}

	conn.State = domain.StateResolvingDNS
	return nil
}

func (s *Server) finishTargetConnect(conn *domain.Connection) error {
	fd := conn.TargetFD

	nerr, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_ERROR)
	if err != nil {
		return err
	}
	if nerr != 0 {
		return syscall.Errno(nerr)
	}

	bndAddr, bndPort, err := network.GetLocalIp4Addr(fd)
	if err != nil {
		return err
	}

	resp := make([]byte, 10)
	resp[0] = domain.SocksVersion5
	resp[1] = 0x00
	resp[2] = 0x00
	resp[3] = domain.SocksAddrTypeIPv4
	copy(resp[4:8], bndAddr[:])
	binary.BigEndian.PutUint16(resp[8:10], uint16(bndPort))

	if _, err := network.Write(conn.ClientFD, resp); err != nil {
		return err
	}

	conn.State = domain.StateRelaying

	clientMask := netpoll.EventRead | netpoll.EventError | netpoll.EventHup
	if len(conn.ToClientBuf) > 0 {
		clientMask |= netpoll.EventWrite
	}
	_ = s.poller.Mod(conn.ClientFD, clientMask)

	targetMask := netpoll.EventRead | netpoll.EventError | netpoll.EventHup
	if len(conn.ToTargetBuf) > 0 {
		targetMask |= netpoll.EventWrite
	}
	_ = s.poller.Mod(conn.TargetFD, targetMask)

	return nil
}

func (s *Server) handleClientWritable(conn *domain.Connection) error {
	if len(conn.ToClientBuf) == 0 {
		s.updateClientPoller(conn)
		return nil
	}
	return s.flushToClient(conn)
}

func (s *Server) handleTargetWritable(conn *domain.Connection) error {
	if len(conn.ToTargetBuf) == 0 {
		s.updateTargetPoller(conn)
		return nil
	}
	return s.flushToTarget(conn)
}

func (s *Server) handleTargetReadable(conn *domain.Connection) error {

	buf := make([]byte, 4096)

	n, err := network.Read(conn.TargetFD, buf)
	if n > 0 {
		conn.ToClientBuf = append(conn.ToClientBuf, buf[:n]...)
	}

	if err != nil {
		if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) {
			return nil
		}
		if errors.Is(err, io.EOF) && n == 0 {
			_ = unix.Shutdown(conn.ClientFD, unix.SHUT_WR)
			_ = unix.Shutdown(conn.TargetFD, unix.SHUT_RD)
			conn.ClientWriteClosed = true
			conn.TargetReadClosed = true
			if conn.ClientReadClosed && conn.TargetWriteClosed {
				s.closeConnection(conn)
			}
			return nil
		}
		return err
	}

	if n == 0 {
		_ = unix.Shutdown(conn.ClientFD, unix.SHUT_WR)
		_ = unix.Shutdown(conn.TargetFD, unix.SHUT_RD)
		conn.ClientWriteClosed = true
		conn.TargetReadClosed = true
		if conn.ClientReadClosed && conn.TargetWriteClosed {
			s.closeConnection(conn)
		}
	}

	if len(conn.ToClientBuf) > 0 {
		if err := s.flushToClient(conn); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) flushToTarget(conn *domain.Connection) error {
	for len(conn.ToTargetBuf) > 0 {
		n, err := network.Write(conn.TargetFD, conn.ToTargetBuf)
		if n > 0 {
			conn.ToTargetBuf = conn.ToTargetBuf[n:]
		}

		if err != nil {
			if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) {
				break
			}
			return err
		}

		if n == 0 {
			break
		}
	}

	s.updateTargetPoller(conn)

	return nil
}

func (s *Server) flushToClient(conn *domain.Connection) error {
	for len(conn.ToClientBuf) > 0 {
		n, err := network.Write(conn.ClientFD, conn.ToClientBuf)
		if n > 0 {
			conn.ToClientBuf = conn.ToClientBuf[n:]
		}

		if err != nil {
			if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) {
				break
			}
			return err
		}

		if n == 0 {
			break
		}
	}

	s.updateClientPoller(conn)
	return nil
}

func (s *Server) updateClientPoller(conn *domain.Connection) {
	mask := netpoll.EventRead | netpoll.EventError | netpoll.EventHup
	if len(conn.ToClientBuf) > 0 {
		mask |= netpoll.EventWrite
	}
	_ = s.poller.Mod(conn.ClientFD, mask)
}

func (s *Server) updateTargetPoller(conn *domain.Connection) {
	mask := netpoll.EventRead | netpoll.EventError | netpoll.EventHup
	if len(conn.ToTargetBuf) > 0 {
		mask |= netpoll.EventWrite
	}
	_ = s.poller.Mod(conn.TargetFD, mask)
}

func (s *Server) closeConnection(conn *domain.Connection) {
	slog.Info("connection fully closed",
		"connID", conn.ID,
		"reason", "both sides EOF",
	)
	_ = s.poller.Del(conn.ClientFD)
	if conn.TargetFD >= 0 {
		_ = s.poller.Del(conn.TargetFD)
	}

	_ = network.Close(conn.ClientFD)
	if conn.TargetFD >= 0 {
		_ = network.Close(conn.TargetFD)
	}
	s.connManager.DeleteConnection(conn)
}
