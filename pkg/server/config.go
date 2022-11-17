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
	if config.Host != "" {
		c.Host = config.Host
	}
	if config.Auth != "" {
		c.Auth = config.Auth
	}
	if config.Path != "" {
		c.Path = config.Path
	}
	if config.Port != 0 {
		c.Port = config.Port
	}
}
