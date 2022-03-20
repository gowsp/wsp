package client

import (
	"flag"
	"strings"

	"github.com/gowsp/wsp/internal/cmd"
)

type sliceArg []string

func (i *sliceArg) String() string {
	return "slice agrs"
}

func (i *sliceArg) Set(value string) error {
	args := strings.Split(value, " ")
	*i = append(*i, args...)
	return nil
}

type Config struct {
	Auth    string   `json:"auth,omitempty"`
	Server  string   `json:"server,omitempty"`
	Local   sliceArg `json:"local,omitempty"`
	Remote  sliceArg `json:"remote,omitempty"`
	Dynamic sliceArg `json:"dynamic,omitempty"`
}

func (c *Config) NewEmpty() cmd.CmdConfig {
	return &Config{}
}
func (c *Config) Merge(conf cmd.CmdConfig) {
	config := conf.(*Config)
	if c.Server == "" {
		c.Server = config.Server
	}
	if c.Auth == "" {
		c.Auth = config.Auth
	}
	c.Local = append(c.Local, config.Local...)
	c.Remote = append(c.Remote, config.Remote...)
	c.Dynamic = append(c.Dynamic, config.Dynamic...)
}

func (config *Config) BindFlag() {
	flag.StringVar(&config.Server, "s", "", "wsps server url")
	flag.StringVar(&config.Auth, "t", "", "wsps auth token")
	flag.Var(&config.Local, "L", `Specifies that connections to the given TCP port
on the local (client) host are to be forwarded to
the given host and port, on the remote side.  This 
works by allocating a socket to listen to either a TCP port on the local side
protocols://remote_channel[:password]@[bind_address]:port`)
	flag.Var(&config.Remote, "R", `Specifies that connections to the given TCP port or Unix
socket on the remote (server) host are to be forwarded to the local side
tcp://bind_address:port
tunnel://channel[:password]@
protocols://bind_address:port/[path]?mode=[mode]&value=[value]`)
	flag.Var(&config.Dynamic, "D", `Specifies a local “dynamic” application-level port
forwarding.  This works by allocating a socket to listen to
port on the local side, http and socks5 protocols are supported
protocols://remote_channel[:password]@[bind_address]:port`)
}
