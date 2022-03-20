package server

import (
	"flag"
	"strings"

	"github.com/gowsp/wsp/internal/cmd"
)

type Config struct {
	Host string `json:"host,omitempty"`
	Auth string `json:"auth,omitempty"`
	Path string `json:"path,omitempty"`
	Port uint64 `json:"port,omitempty"`
}

func (c *Config) clean() {
	c.Path = strings.TrimPrefix(c.Path, "/")
	c.Path = strings.TrimSpace(c.Path)
}

func (c *Config) NewEmpty() cmd.CmdConfig {
	return &Config{}
}
func (c *Config) Merge(conf cmd.CmdConfig) {
	config := conf.(*Config)
	if c.Host == "" {
		c.Host = config.Host
	}
	if c.Auth == "" {
		c.Auth = config.Auth
	}
	if c.Path == "" {
		c.Path = config.Path
	}
	if c.Port == 0 {
		c.Port = config.Port
	}
}

func (config *Config) BindFlag() {
	flag.StringVar(&config.Host, "h", "", "wsps domain name")
	flag.StringVar(&config.Auth, "t", "", "wsps auth token")
	flag.StringVar(&config.Path, "path", "/", "wsps websocket path")
	flag.Uint64Var(&config.Port, "p", 8080, `Specifies the port on which the websocket server listens for connections`)
}
