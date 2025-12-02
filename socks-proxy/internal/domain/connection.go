package domain

type State int

const (
	StateGreeting State = iota
	StateRequest
	StateResolvingDNS
	StateConnectingTarget
	StateRelaying
	StateClosing
)

type Connection struct {
	ID int
	
	ClientFD int
	TargetFD int

	State State

	ClientReaderBuf []byte
	ToTargetBuf     []byte

	TargetReaderBuf []byte
	ToClientBuf     []byte

	ClientReadClosed  bool
	ClientWriteClosed bool
	TargetReadClosed  bool
	TargetWriteClosed bool

	PendingDomain string
	PendingPort   uint16
	DNSQueryID    uint16
}
