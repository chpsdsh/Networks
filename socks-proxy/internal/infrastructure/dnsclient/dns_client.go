package dnsclient

import (
	"errors"

	"github.com/miekg/dns"
)

type Client struct {
	ResolverIp   [4]byte
	ResolverPort int
}

func NewClient(resolverIp [4]byte, resolverPort int) *Client {
	return &Client{ResolverIp: resolverIp, ResolverPort: resolverPort}
}

func (c *Client) BuildQuery(domain string) (uint16, []byte, error) {
	msg := new(dns.Msg)
	msg.Id = dns.Id()
	msg.RecursionDesired = true
	msg.Question = []dns.Question{
		{
			Name:   dns.Fqdn(domain),
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET,
		},
	}
	raw, err := msg.Pack()
	if err != nil {
		return 0, nil, err
	}
	return msg.Id, raw, nil
}

func (c *Client) ParseResponse(packet []byte) (uint16, [4]byte, error) {
	var zero [4]byte

	var msg dns.Msg
	if err := msg.Unpack(packet); err != nil {
		return 0, zero, err
	}

	for _, ans := range msg.Answer {
		if a, ok := ans.(*dns.A); ok {
			ip4 := a.A.To4()
			if ip4 == nil {
				continue
			}
			var out [4]byte
			copy(out[:], ip4)
			return msg.Id, out, nil
		}
	}

	return msg.Id, zero, errors.New("no A record in DNS response")
}
