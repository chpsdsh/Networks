package netpoll

import (
	"errors"

	"golang.org/x/sys/unix"
)

type EventMask uint32

const (
	EventRead EventMask = 1 << iota
	EventWrite
	EventError
	EventHup
)

type Event struct {
	Events EventMask
	FD     int
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

func (p *Poller) Wait(maxEvents int, timeoutMs int) ([]Event, error) {
	if maxEvents <= 0 {
		maxEvents = 1
	}
	rawEvents := make([]unix.EpollEvent, maxEvents)
	n, err := unix.EpollWait(p.fd, rawEvents, timeoutMs)
	if err != nil {
		if errors.Is(err, unix.EINTR) {
			return nil, nil
		}
		return nil, err
	}
	events := make([]Event, 0, n)
	for i := 0; i < n; i++ {
		re := rawEvents[i]
		events = append(events, Event{
			Events: epollToMask(re.Events),
			FD:     int(re.Fd),
		})
	}
	return events, nil
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

func epollToMask(ev uint32) EventMask {
	var mask EventMask
	if ev&unix.EPOLLIN != 0 {
		mask |= EventRead
	}
	if ev&unix.EPOLLOUT != 0 {
		mask |= EventWrite
	}
	if ev&unix.EPOLLHUP != 0 {
		mask |= EventHup
	}
	if ev&unix.EPOLLERR != 0 {
		mask |= EventError
	}
	return mask
}
