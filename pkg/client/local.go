package client

import (
	"io"
	"log"
	"net"

	"github.com/gowsp/wsp/pkg/msg"
)

func (c *Wspc) LocalForward() {
	for _, val := range c.config.Local {
		conf, err := msg.NewWspConfig(msg.WspType_LOCAL, val)
		if err != nil {
			log.Println("forward local error,", err)
			continue
		}
		go c.ListenLocal(conf)
	}
}
func (c *Wspc) ListenLocal(conf *msg.WspConfig) {
	log.Println("listen local on channel", conf.Channel())
	local, err := net.Listen(conf.Network(), conf.Address())
	if err != nil {
		log.Println(err)
		return
	}
	for {
		conn, err := local.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		c.NewLocalConn(conn, conf)
	}
}

func (c *Wspc) NewLocalConn(local net.Conn, config *msg.WspConfig) {
	channel := config.Channel()
	log.Println("open remote channel", channel)
	remote, err := c.wan.DialTCP(local, config)
	if err != nil {
		local.Close()
		log.Println("close local", local.LocalAddr(), err.Error())
		return
	}
	io.Copy(remote, local)
	remote.Close()
	log.Println("close local", local.LocalAddr())
}
