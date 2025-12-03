package domain

const (
	SocksNoAuth                 = 0x00
	SocksNoAccessibleAuthMethod = 0xFF
	SocksVersion5               = 0x05
	SocksEstablishTcpConnection = 0x01
	SocksAddrTypeIPv4           = 0x01
	SocksAddrTypeDomain         = 0x03
	SocksAddrTypeIPv6           = 0x04
	SocksCommandNotSupported    = 0x07
	SocksAddrTypeNotSupported   = 0x08
)
