package server

import (
	"net"
	"time"

	"github.com/gowsp/wsp/pkg/logger"
	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/stream"
)

func (c *conn) NewDynamic(id string, conf *msg.WspConfig) error {
	address := conf.Address()
	logger.Info("open connect %s", address)

	remote, err := net.DialTimeout(conf.Network(), address, 5*time.Second)
	if err != nil {
		return err
	}
	local, err := c.wan.Accept(id, remote.RemoteAddr(), conf)
	if err != nil {
		return err
	}
	stream.Copy(local, remote)
	logger.Info("close connect %s, addr %s", address, remote.RemoteAddr())
	return nil
}
