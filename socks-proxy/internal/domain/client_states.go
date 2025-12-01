package domain

type ClientStateType int

const (
	StateGreeting ClientStateType = iota
	StateGreetingResponse
	StateRequest
	StateResolvingDNS
	StateConnectingTarget
	StateRelaying
	StateClosing
)
