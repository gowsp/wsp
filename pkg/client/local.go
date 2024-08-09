package client

import (
	"net"

	"github.com/gowsp/wsp/pkg/logger"
	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/stream"
)

func (c *Wspc) LocalForward() {
	for _, val := range c.config.Local {
		conf, err := msg.NewWspConfig(msg.WspType_LOCAL, val)
		if err != nil {
			logger.Error("local config %s error: %s", val, err)
			continue
		}
		go c.ListenLocal(conf)
	}
}
func (c *Wspc) ListenLocal(conf *msg.WspConfig) {
	channel := conf.Channel()
	logger.Info("listen local on channel %s", channel)
	local, err := net.Listen(conf.Network(), conf.Address())
	if err != nil {
		logger.Error("listen local channel %s, error: %s", channel, err)
		return
	}
	for {
		conn, err := local.Accept()
		if err != nil {
			logger.Error("accept local channel %s, error: %s", channel, err)
			continue
		}
		go c.NewLocalConn(conn, conf)
	}
}

func (c *Wspc) NewLocalConn(local net.Conn, config *msg.WspConfig) {
	channel := config.Channel()
	logger.Info("open remote channel %s", channel)
	remote, err := c.wan.DialTCP(local.LocalAddr(), config)
	if err != nil {
		local.Close()
		logger.Error("close local channel %s, addr %s, error: %s", channel, local.LocalAddr(), err)
		return
	}
	stream.Copy(local, remote)
	logger.Info("close local channel %s, addr %s", channel, local.LocalAddr())
}
