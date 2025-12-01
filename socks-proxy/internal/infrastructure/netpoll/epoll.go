package netpoll

import "golang.org/x/sys/unix"

type EventMask uint32

const (
	EventRead EventMask = 1 << iota
	EventWrite
	EventError
	EventHup
)

type Event struct {
	Mask EventMask
	FD   int
}

type Poller struct {
	fd int
}

func NewPoller() (*Poller, error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	return &Poller{fd: fd}, nil
}

func (p *Poller) Close() error {
	if p == nil || p.fd <= 0 {
		return nil
	}
	return unix.Close(p.fd)
}

func (p *Poller) Add(fd int, mask EventMask) error {
	ev := &unix.EpollEvent{
		Events: maskToEpoll(mask),
		Fd:     int32(fd),
	}
	if err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, ev); err != nil {
		return err
	}
	return nil
}

func (p *Poller) Mod(fd int, mask EventMask) error {
	ev := &unix.EpollEvent{
		Events: maskToEpoll(mask),
		Fd:     int32(fd),
	}
	if err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_MOD, fd, ev); err != nil {
		return err
	}
	return nil
}

func (p *Poller) Del(fd int) error {
	if err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_DEL, fd, nil); err != nil {
		return err
	}
	return nil
}

func maskToEpoll(mask EventMask) uint32 {
	var ev uint32
	if mask&EventRead != 0 {
		ev |= unix.EPOLLIN
	}
	if mask&EventWrite != 0 {
		ev |= unix.EPOLLOUT
	}
	return ev
}
