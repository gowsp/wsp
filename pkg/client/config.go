package client

import (
	"strings"

	"github.com/gowsp/wsp/pkg/logger"
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
	Log    logger.Config `json:"log,omitempty"`
	Client []WspcConfig  `json:"client,omitempty"`
}

type WspcConfig struct {
	Auth    string   `json:"auth,omitempty"`
	Server  string   `json:"server,omitempty"`
	Local   sliceArg `json:"local,omitempty"`
	Remote  sliceArg `json:"remote,omitempty"`
	Dynamic sliceArg `json:"dynamic,omitempty"`
}
