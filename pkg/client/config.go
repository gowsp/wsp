package client

import (
	"net"
	"strconv"
)

type Config struct {
	Auth   string `json:"auth,omitempty"`
	Server string `json:"server,omitempty"`
	Socks5 string `json:"socks5,omitempty"`
	Addrs  []Addr `json:"addrs,omitempty"`
}
type Addr struct {
	Name      string `json:"name,omitempty"`
	Secret    string `json:"secret,omitempty"`
	Forward   string `json:"forward,omitempty"`
	Remote    string `json:"remote,omitempty"`
	LocalAddr string `json:"local_addr,omitempty"`
	LocalPort int    `json:"local_port,omitempty"`
}

func (a *Addr) Address() string {
	return net.JoinHostPort(a.LocalAddr, strconv.Itoa(a.LocalPort))
}
