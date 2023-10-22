package server

import (
	"io"
	"net"
	"time"

	"github.com/gowsp/wsp/pkg/logger"
	"github.com/gowsp/wsp/pkg/msg"
)

func (c *conn) NewDynamic(id string, conf *msg.WspConfig) error {
	address := conf.Address()
	logger.Info("open connect %s", address)

	remote, err := net.DialTimeout(conf.Network(), address, 5*time.Second)
	if err != nil {
		return err
	}
	local, err := c.wan.Accept(id, remote, conf)
	if err != nil {
		return err
	}
	io.Copy(local, remote)
	local.Close()
	logger.Info("close connect %s, addr %s", address, remote.RemoteAddr())
	return nil
}
