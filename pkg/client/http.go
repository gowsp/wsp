package client

import (
	"log"
	"net"
	"time"

	"github.com/gowsp/wsp/pkg"
)

func (c *Wspc) NewHttpConn(id, address string) {
	log.Println("recive http reqeust", address)
	val, ok := c.network.Load(address)
	if !ok {
		c.wan.CloseRemote(id, address+" not found")
		return
	}
	addr := val.(Addr)
	conn, err := net.DialTimeout("tcp", addr.Address(), 5*time.Second)
	if err != nil {
		c.wan.Reply(id, false)
		return
	}

	in := c.wan.NewWriter(id)
	repeater := pkg.NewNetRepeater(in, conn)
	c.routing.AddRepeater(id, repeater)
	defer c.routing.Delete(id)

	c.wan.Reply(id, true)

	repeater.Copy()
	log.Println("close http connect", address)
}
