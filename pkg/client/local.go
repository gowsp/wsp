package client

import (
	"log"
	"net"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/proxy"
	"github.com/segmentio/ksuid"
)

func (c *Wspc) LocalForward() {
	for _, val := range c.Config.Local {
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

func (c *Wspc) NewLocalConn(conn net.Conn, conf *msg.WspConfig) {
	channel := conf.Channel()
	log.Println("open remote channel", channel)
	id := ksuid.New().String()

	c.routing.AddPending(id, &proxy.Pending{OnReponse: func(data *msg.Data, message *msg.WspResponse) {
		if message.Code == msg.WspCode_FAILED {
			log.Printf("close channel %s, %s", channel, message.Data)
			conn.Close()
			return
		}
		in := c.wan.NewWriter(id)
		repeater := proxy.NewNetRepeater(in, conn)
		c.routing.AddRepeater(id, repeater)
		defer c.routing.Delete(id)
		repeater.Copy()
		log.Println("close remote channel", channel)
	}})

	if err := c.wan.Dail(id, conf); err != nil {
		c.routing.DeleteConn(id)
		log.Println(err)
		conn.Close()
	}
}
