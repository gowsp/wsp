package server

import (
	"strings"

	"github.com/gowsp/wsp/pkg/logger"
)

type Config struct {
	Log logger.Config `json:"log,omitempty"`

	SSL  SSL    `json:"ssl,omitempty"`
	Host string `json:"host,omitempty"`
	Auth string `json:"auth,omitempty"`
	Path string `json:"path,omitempty"`
	Port uint64 `json:"port,omitempty"`
}

type SSL struct {
	Enable bool   `json:"enable,omitempty"`
	Key    string `json:"key,omitempty"`
	Cert   string `json:"cert,omitempty"`
}

func (c *Config) EnbleSSL() bool {
	return c.SSL.Enable && c.SSL.Key != "" && c.SSL.Cert != ""
}
func (c *Config) clean() {
	c.Path = strings.TrimPrefix(c.Path, "/")
	c.Path = strings.TrimSpace(c.Path)
}
