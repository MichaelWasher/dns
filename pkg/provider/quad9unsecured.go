package provider

import (
	"net"
	"net/url"
)

type quad9Unsecured struct{}

func Quad9Unsecured() Provider {
	return &quad9Unsecured{}
}

func (q *quad9Unsecured) String() string {
	return "Quad9 Unsecured"
}

func (q *quad9Unsecured) DNS() DNSServer {
	return DNSServer{
		IPv4: []net.IP{{9, 9, 9, 10}, {149, 112, 112, 10}},
		IPv6: []net.IP{
			{0x26, 0x20, 0x0, 0xfe, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x10},
			{0x26, 0x20, 0x0, 0xfe, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xfe, 0x0, 0x10},
		},
	}
}

func (q *quad9Unsecured) DoT() DoTServer {
	return DoTServer{
		IPv4: []net.IP{{9, 9, 9, 9}, {149, 112, 112, 9}},
		IPv6: []net.IP{
			{0x26, 0x20, 0x0, 0xfe, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x9},
			{0x26, 0x20, 0x0, 0xfe, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xfe, 0x0, 0x9},
		},
		Name: "dns10.quad9.net",
		Port: defaultDoTPort,
	}
}

func (q *quad9Unsecured) DoH() DoHServer {
	// See https://developers.quad9.com/speed/public-dns/docs/doh
	return DoHServer{
		URL: &url.URL{
			Scheme: "https",
			Host:   "dns10.quad9.net",
			Path:   "/dns-query",
		},
	}
}