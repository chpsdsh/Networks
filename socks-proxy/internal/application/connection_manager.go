package application

import (
	"socks-proxy/internal/domain"
)

type ConnectionManager struct {
	nextId       int
	byClientFd   map[int]*domain.Connection
	byTargetFd   map[int]*domain.Connection
	byDnsQueryFd map[uint16]*domain.Connection
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		byClientFd:   make(map[int]*domain.Connection),
		byTargetFd:   make(map[int]*domain.Connection),
		byDnsQueryFd: make(map[uint16]*domain.Connection),
	}
}

func (m *ConnectionManager) newId() int {
	m.nextId++
	return m.nextId
}

func (m *ConnectionManager) NewClientConnection(clientFd int) *domain.Connection {
	c := &domain.Connection{
		ID:       m.newId(),
		ClientFD: clientFd,
		TargetFD: -1,
		State:    domain.StateGreeting,
	}
	m.byClientFd[clientFd] = c
	return c
}

func (m *ConnectionManager) GetClientConnectionByFd(fd int) *domain.Connection {
	return m.byClientFd[fd]
}

func (m *ConnectionManager) GetTargetConnectionByFd(fd int) *domain.Connection {
	return m.byTargetFd[fd]
}

func (m *ConnectionManager) AttachTarget(conn *domain.Connection, targetFd int) {
	conn.TargetFD = targetFd
	m.byTargetFd[targetFd] = conn
}

func (m *ConnectionManager) GetDnsConnectionById(id uint16) *domain.Connection {
	return m.byDnsQueryFd[id]
}

func (m *ConnectionManager) BindDNSQuery(conn *domain.Connection, id uint16) {
	conn.DNSQueryID = id
	m.byDnsQueryFd[id] = conn
}

func (m *ConnectionManager) DeleteConnection(conn *domain.Connection) {
	delete(m.byClientFd, conn.ClientFD)
	if conn.TargetFD >= 0 {
		delete(m.byTargetFd, conn.TargetFD)
	}
	if conn.DNSQueryID != 0 {
		delete(m.byDnsQueryFd, conn.DNSQueryID)
	}
}
